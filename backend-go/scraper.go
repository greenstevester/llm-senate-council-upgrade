package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	// Base URL for bills before parliament
	BillsBaseURL = "https://www.aph.gov.au/Parliamentary_Business/Bills_Legislation/Bills_before_Parliament"

	// HTTP timeout for each request
	ScraperTimeout = 30 * time.Second

	// Delay between page requests to be respectful
	PageRequestDelay = 500 * time.Millisecond

	// User agent for HTTP requests
	UserAgent = "LLM-Council-Bills-Scraper/1.0 (Educational Project)"
)

// Bill represents a single parliamentary bill
type Bill struct {
	ID                 string    `json:"id"`                   // e.g., "r7365", "s1254"
	Title              string    `json:"title"`
	DateIntroduced     string    `json:"date_introduced"`      // e.g., "03 Sep 2025"
	Chamber            string    `json:"chamber"`              // "Senate" or "House of Representatives"
	Status             string    `json:"status"`               // e.g., "Before Senate"
	PortfolioSponsor   string    `json:"portfolio_sponsor"`    // e.g., "Attorney-General"
	Summary            string    `json:"summary"`
	BillURL            string    `json:"bill_url"`             // ParlInfo link
	ExplanatoryMemoURL string    `json:"explanatory_memo_url"` // ParlInfo link
	ScrapedAt          time.Time `json:"scraped_at"`
}

// BillsResponse represents the paginated response
type BillsResponse struct {
	Bills       []Bill    `json:"bills"`
	CurrentPage int       `json:"current_page"`
	TotalPages  int       `json:"total_pages"`
	HasNextPage bool      `json:"has_next_page"`
	LastUpdated time.Time `json:"last_updated"`
}

// FetchBillsPage fetches a single page of bills from the APH website
// Returns the bills found on that page and whether there's a next page
func FetchBillsPage(ctx context.Context, pageNum int) ([]Bill, bool, error) {
	// Construct URL with page parameter
	url := BillsBaseURL
	if pageNum > 1 {
		url = fmt.Sprintf("%s?page=%d&drt=2&drv=7", BillsBaseURL, pageNum)
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: ScraperTimeout,
	}

	// Execute request with retry logic
	var resp *http.Response
	maxRetries := 2
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err = client.Do(req)
		if err == nil {
			break
		}

		if attempt < maxRetries-1 {
			log.Printf("Attempt %d failed, retrying in 2s: %v", attempt+1, err)
			time.Sleep(2 * time.Second)
		}
	}

	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch page %d after %d attempts: %w", pageNum, maxRetries, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("unexpected status code %d for page %d", resp.StatusCode, pageNum)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Parse bills from HTML
	bills, err := ParseBillsHTML(doc)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse bills: %w", err)
	}

	// Check for next page
	hasNext := HasNextPage(doc)

	log.Printf("Fetched page %d: found %d bills, hasNext=%v", pageNum, len(bills), hasNext)

	return bills, hasNext, nil
}

// ParseBillsHTML extracts bill information from the HTML document
func ParseBillsHTML(doc *goquery.Document) ([]Bill, error) {
	var bills []Bill
	scrapedAt := time.Now()

	// Find all bill entries - they're in divs with h4 headers
	doc.Find("h4").Each(func(i int, s *goquery.Selection) {
		// Check if this h4 contains a bill title (has a link)
		titleLink := s.Find("a").First()
		if titleLink.Length() == 0 {
			return // Skip if no link found
		}

		// Extract bill ID from link href
		href, exists := titleLink.Attr("href")
		if !exists {
			return
		}

		billID := extractBillID(href)
		if billID == "" {
			return // Skip if can't extract ID
		}

		// Extract title
		title := strings.TrimSpace(titleLink.Text())
		if title == "" {
			return // Skip if no title
		}

		// Navigate to parent container to find other bill details
		container := s.Parent()

		// Extract bill details from the container
		var dateIntroduced, chamber, status, portfolioSponsor, summary string
		var billURL, memoURL string

		// Look for bill metadata in paragraphs after the h4
		container.Find("p").Each(func(j int, p *goquery.Selection) {
			text := strings.TrimSpace(p.Text())

			// Check if this paragraph contains date/chamber/status info
			if strings.Contains(text, "Introduced:") || strings.Contains(text, "Senate") || strings.Contains(text, "House of Representatives") {
				parts := strings.Split(text, "|")
				for _, part := range parts {
					part = strings.TrimSpace(part)

					// Extract date (format: "03 Sep 2025")
					if matched, _ := regexp.MatchString(`\d{2}\s+\w{3}\s+\d{4}`, part); matched {
						dateIntroduced = part
					}

					// Extract chamber
					if strings.Contains(part, "Senate") {
						chamber = "Senate"
					} else if strings.Contains(part, "House of Representatives") {
						chamber = "House of Representatives"
					}

					// Extract status (typically "Before Senate" or "Before House")
					if strings.HasPrefix(part, "Before ") {
						status = part
					}
				}
			}

			// Check for portfolio/sponsor
			if strings.Contains(text, "Portfolio:") || strings.Contains(text, "Sponsor:") {
				// Extract the text after the colon
				if idx := strings.Index(text, ":"); idx != -1 {
					portfolioSponsor = strings.TrimSpace(text[idx+1:])
				}
			}

			// Extract summary (usually a longer paragraph without metadata keywords)
			if !strings.Contains(text, "Introduced:") &&
			   !strings.Contains(text, "Portfolio:") &&
			   !strings.Contains(text, "Sponsor:") &&
			   !strings.Contains(text, "|") &&
			   len(text) > 50 {
				if summary == "" || len(text) > len(summary) {
					summary = text
				}
			}
		})

		// Extract links (Bill and Explanatory Memorandum)
		container.Find("a").Each(func(j int, a *goquery.Selection) {
			linkText := strings.TrimSpace(a.Text())
			linkHref, _ := a.Attr("href")

			if strings.Contains(linkText, "Bill") && !strings.Contains(linkText, "Explanatory") {
				billURL = normalizeURL(linkHref)
			}
			if strings.Contains(linkText, "Explanatory Memorandum") {
				memoURL = normalizeURL(linkHref)
			}
		})

		// Create bill object
		bill := Bill{
			ID:                 billID,
			Title:              title,
			DateIntroduced:     dateIntroduced,
			Chamber:            chamber,
			Status:             status,
			PortfolioSponsor:   portfolioSponsor,
			Summary:            summary,
			BillURL:            billURL,
			ExplanatoryMemoURL: memoURL,
			ScrapedAt:          scrapedAt,
		}

		bills = append(bills, bill)
	})

	return bills, nil
}

// extractBillID extracts the bill ID from a URL
// Expected format: /Parliamentary_Business/Bills_Legislation/bd/bd1234
func extractBillID(href string) string {
	// Look for patterns like "bd/bd1234" or just the ID part
	re := regexp.MustCompile(`bd/bd(\w+)`)
	matches := re.FindStringSubmatch(href)
	if len(matches) > 1 {
		return matches[1]
	}

	// Alternative: extract from last segment
	parts := strings.Split(href, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// Remove "bd" prefix if present
		return strings.TrimPrefix(lastPart, "bd")
	}

	return ""
}

// normalizeURL ensures URLs are absolute
func normalizeURL(href string) string {
	if href == "" {
		return ""
	}

	// If already absolute, return as-is
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	// Make relative URLs absolute
	if strings.HasPrefix(href, "/") {
		return "https://www.aph.gov.au" + href
	}

	return "https://www.aph.gov.au/" + href
}

// HasNextPage checks if there's a next page link in the pagination
func HasNextPage(doc *goquery.Document) bool {
	// Look for pagination links
	hasNext := false
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if strings.Contains(strings.ToLower(text), "next") {
			hasNext = true
		}
	})
	return hasNext
}

// ExtractPaginationInfo extracts current page and total pages from HTML
func ExtractPaginationInfo(doc *goquery.Document) (currentPage int, totalPages int, hasNext bool) {
	currentPage = 1
	totalPages = 1
	hasNext = false

	// Look for pagination info
	doc.Find(".pagination, nav[aria-label='Pagination']").Each(func(i int, s *goquery.Selection) {
		// Find active page number
		s.Find(".active, .current").Each(func(j int, active *goquery.Selection) {
			pageText := strings.TrimSpace(active.Text())
			if num, err := strconv.Atoi(pageText); err == nil {
				currentPage = num
			}
		})

		// Find all page numbers to determine total
		s.Find("a").Each(func(j int, link *goquery.Selection) {
			pageText := strings.TrimSpace(link.Text())
			if num, err := strconv.Atoi(pageText); err == nil && num > totalPages {
				totalPages = num
			}

			// Check for "Next" link
			if strings.Contains(strings.ToLower(pageText), "next") {
				hasNext = true
			}
		})
	})

	return currentPage, totalPages, hasNext
}

// FetchAllBills fetches all bills across all pages
func FetchAllBills(ctx context.Context) ([]Bill, error) {
	var allBills []Bill
	pageNum := 1

	log.Println("Starting to fetch all bills from APH website...")

	for {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Fetch page
		bills, hasNext, err := FetchBillsPage(ctx, pageNum)
		if err != nil {
			// Log error but continue with what we have
			log.Printf("Error fetching page %d: %v", pageNum, err)
			if pageNum == 1 {
				// If first page fails, return error
				return nil, fmt.Errorf("failed to fetch first page: %w", err)
			}
			// Otherwise, return bills we've collected so far
			break
		}

		// Add bills to collection
		allBills = append(allBills, bills...)

		// Check if there are more pages
		if !hasNext {
			log.Printf("Reached last page. Total bills collected: %d", len(allBills))
			break
		}

		// Increment page number
		pageNum++

		// Rate limiting: wait before next request
		time.Sleep(PageRequestDelay)
	}

	return allBills, nil
}

// CalculateTotalPages estimates total pages based on bill count
// Assumes roughly 20 bills per page
func CalculateTotalPages(billCount int) int {
	if billCount == 0 {
		return 1
	}
	pages := billCount / 20
	if billCount%20 > 0 {
		pages++
	}
	return pages
}
