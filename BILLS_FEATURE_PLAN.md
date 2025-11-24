# Bills Before Parliament Feature - Implementation Plan

## Overview

Add a new feature to display Australian Parliamentary bills in a scrollable sidebar panel, fetched from the Australian Parliament House website with full pagination support.

## Architecture Analysis

### Current System
- **Backend**: Go (Gin framework) on port 8001
- **Frontend**: React + Vite with light theme (#4a90e2 primary)
- **Layout**: Sidebar (260px) + Main chat interface
- **Data Flow**: RESTful JSON API + SSE streaming for council deliberations

### Target Website Analysis
**URL**: `https://www.aph.gov.au/Parliamentary_Business/Bills_Legislation/Bills_before_Parliament`

**Structure**:
- Bills displayed as `<div>` elements with `<h4>` headers (not tables)
- Pagination via numbered links with querystring params (`?page=2&drt=2&drv=7...`)
- Bill identifiers: alphanumeric codes (`r7365` for House, `s1254` for Senate)

**Bill Information**:
- Title (clickable link in `<h4>`)
- Date introduced (e.g., "03 Sep 2025")
- Chamber (Senate/House of Representatives)
- Status (e.g., "Before Senate")
- Portfolio/Sponsor
- Summary paragraph
- Links: "Bill" and "Explanatory Memorandum" (ParlInfo URLs)
- Track option

## Implementation Plan

### Phase 1: Backend - Web Scraping Service

#### 1.1 Create `scraper.go` Module
**Location**: `backend-go/scraper.go`

**Structs**:
```go
// Bill represents a single parliamentary bill
type Bill struct {
    ID                string    `json:"id"`                 // e.g., "r7365", "s1254"
    Title             string    `json:"title"`
    DateIntroduced    string    `json:"date_introduced"`    // e.g., "03 Sep 2025"
    Chamber           string    `json:"chamber"`            // "Senate" or "House of Representatives"
    Status            string    `json:"status"`             // e.g., "Before Senate"
    PortfolioSponsor  string    `json:"portfolio_sponsor"`  // e.g., "Attorney-General"
    Summary           string    `json:"summary"`
    BillURL           string    `json:"bill_url"`           // ParlInfo link
    ExplanatoryMemoURL string   `json:"explanatory_memo_url"` // ParlInfo link
    ScrapedAt         time.Time `json:"scraped_at"`
}

// BillsResponse represents the paginated response
type BillsResponse struct {
    Bills          []Bill `json:"bills"`
    CurrentPage    int    `json:"current_page"`
    TotalPages     int    `json:"total_pages"`
    HasNextPage    bool   `json:"has_next_page"`
    LastUpdated    time.Time `json:"last_updated"`
}
```

**Functions**:
1. `FetchBillsPage(pageNum int) ([]Bill, bool, error)`
   - Takes page number (1-based)
   - Constructs URL with proper querystring
   - HTTP GET with User-Agent header
   - Returns: bills slice, hasNextPage bool, error
   - Timeout: 30 seconds per request

2. `ParseBillsHTML(htmlContent string) ([]Bill, error)`
   - Uses `golang.org/x/net/html` or `github.com/PuerkitoBio/goquery`
   - Locates bill `<div>` containers
   - Extracts all bill fields (title, date, chamber, status, etc.)
   - Parses ParlInfo links
   - Returns structured Bill slice

3. `ExtractPaginationInfo(htmlContent string) (currentPage int, totalPages int, hasNext bool, error)`
   - Parses pagination controls
   - Identifies current page number
   - Determines if "Next" link exists
   - Returns pagination metadata

4. `FetchAllBills() ([]Bill, error)`
   - Starts at page 1
   - Loops through pages following "Next" links
   - Uses goroutines with rate limiting (500ms delay between requests)
   - Collects all bills across pages
   - Returns complete bill list
   - Implements graceful error handling (continues on partial failures)

**Dependencies**:
```bash
go get github.com/PuerkitoBio/goquery
go get golang.org/x/net/html
```

**Error Handling**:
- Network timeouts: Retry once with exponential backoff
- Parse errors: Log and skip malformed bill entries
- Pagination errors: Return bills collected so far
- HTTP errors: Return appropriate error with status code

#### 1.2 Create `cache.go` Module (Optional but Recommended)
**Location**: `backend-go/cache.go`

**Purpose**: Reduce load on APH website, improve response times

**Implementation**:
```go
type BillsCache struct {
    mu          sync.RWMutex
    bills       []Bill
    lastUpdated time.Time
    ttl         time.Duration // e.g., 5 minutes
}

func (c *BillsCache) Get() ([]Bill, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if time.Since(c.lastUpdated) > c.ttl {
        return nil, false // Cache expired
    }
    return c.bills, true
}

func (c *BillsCache) Set(bills []Bill) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.bills = bills
    c.lastUpdated = time.Now()
}
```

**Cache Strategy**:
- TTL: 5 minutes (configurable via environment variable)
- Background refresh: Optional goroutine to refresh cache periodically
- Thread-safe with `sync.RWMutex`

#### 1.3 API Endpoint in `main.go`
**Location**: `backend-go/main.go`

**Route**:
```go
router.GET("/api/bills", getBillsHandler)
```

**Handler Implementation**:
```go
func getBillsHandler(c *gin.Context) {
    // Check cache first (if implemented)
    if cachedBills, ok := billsCache.Get(); ok {
        c.JSON(http.StatusOK, BillsResponse{
            Bills:       cachedBills,
            CurrentPage: 1,
            TotalPages:  calculateTotalPages(len(cachedBills)),
            HasNextPage: false,
            LastUpdated: billsCache.lastUpdated,
        })
        return
    }

    // Fetch fresh data
    ctx := context.Background()
    bills, err := FetchAllBills(ctx)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": fmt.Sprintf("Failed to fetch bills: %v", err),
        })
        return
    }

    // Update cache
    billsCache.Set(bills)

    // Return response
    c.JSON(http.StatusOK, BillsResponse{
        Bills:       bills,
        CurrentPage: 1,
        TotalPages:  calculateTotalPages(len(bills)),
        HasNextPage: false,
        LastUpdated: time.Now(),
    })
}
```

**Query Parameters (Optional Enhancement)**:
- `?refresh=true`: Force cache refresh
- `?chamber=senate|house`: Filter by chamber
- `?status=before_senate|passed`: Filter by status

### Phase 2: Frontend - Bills Panel Component

#### 2.1 Update `App.jsx` Layout
**Location**: `frontend/src/App.jsx`

**Changes**:
1. Add state for bills panel visibility:
```jsx
const [showBillsPanel, setShowBillsPanel] = useState(false);
```

2. Add layout wrapper to accommodate 3-column layout:
   - Sidebar (260px) - existing conversations
   - Main chat interface (flex: 1) - existing
   - Bills panel (320px) - NEW, toggleable

3. Layout structure:
```jsx
<div className="app">
  <Sidebar {...existing props} />
  <ChatInterface {...existing props} />
  {showBillsPanel && <BillsPanel onClose={() => setShowBillsPanel(false)} />}
</div>
```

4. Add toggle button in header or sidebar to show/hide bills panel

#### 2.2 Create `BillsPanel.jsx` Component
**Location**: `frontend/src/components/BillsPanel.jsx`

**Component Structure**:
```jsx
import { useState, useEffect } from 'react';
import './BillsPanel.css';

export default function BillsPanel({ onClose }) {
  const [bills, setBills] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filter, setFilter] = useState({ chamber: 'all', search: '' });
  const [lastUpdated, setLastUpdated] = useState(null);

  useEffect(() => {
    loadBills();
  }, []);

  const loadBills = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('http://localhost:8001/api/bills');
      if (!response.ok) throw new Error('Failed to fetch bills');
      const data = await response.json();
      setBills(data.bills);
      setLastUpdated(new Date(data.last_updated));
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const filteredBills = bills.filter(bill => {
    const matchesChamber = filter.chamber === 'all' ||
                           bill.chamber.toLowerCase().includes(filter.chamber);
    const matchesSearch = bill.title.toLowerCase().includes(filter.search.toLowerCase());
    return matchesChamber && matchesSearch;
  });

  return (
    <div className="bills-panel">
      <div className="bills-panel-header">
        <h2>Bills Before Parliament</h2>
        <button className="close-btn" onClick={onClose}>×</button>
      </div>

      {/* Filter controls */}
      <div className="bills-filters">
        <input
          type="text"
          placeholder="Search bills..."
          value={filter.search}
          onChange={(e) => setFilter({...filter, search: e.target.value})}
        />
        <select
          value={filter.chamber}
          onChange={(e) => setFilter({...filter, chamber: e.target.value})}
        >
          <option value="all">All Chambers</option>
          <option value="senate">Senate</option>
          <option value="house">House of Representatives</option>
        </select>
      </div>

      {/* Refresh button */}
      <div className="bills-actions">
        <button onClick={loadBills} disabled={loading}>
          {loading ? 'Loading...' : 'Refresh'}
        </button>
        {lastUpdated && (
          <span className="last-updated">
            Updated: {lastUpdated.toLocaleTimeString()}
          </span>
        )}
      </div>

      {/* Bills list */}
      <div className="bills-list">
        {loading && <div className="loading">Loading bills...</div>}
        {error && <div className="error">Error: {error}</div>}
        {!loading && !error && filteredBills.length === 0 && (
          <div className="no-bills">No bills found</div>
        )}
        {!loading && !error && filteredBills.map(bill => (
          <BillCard key={bill.id} bill={bill} />
        ))}
      </div>
    </div>
  );
}

function BillCard({ bill }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="bill-card">
      <div className="bill-header" onClick={() => setExpanded(!expanded)}>
        <h3>{bill.title}</h3>
        <span className="expand-icon">{expanded ? '▼' : '▶'}</span>
      </div>

      <div className="bill-meta">
        <span className="chamber-badge">{bill.chamber}</span>
        <span className="status-badge">{bill.status}</span>
      </div>

      <div className="bill-info">
        <div className="info-row">
          <span className="label">Introduced:</span>
          <span>{bill.date_introduced}</span>
        </div>
        <div className="info-row">
          <span className="label">Portfolio:</span>
          <span>{bill.portfolio_sponsor}</span>
        </div>
      </div>

      {expanded && (
        <div className="bill-details">
          <p className="bill-summary">{bill.summary}</p>
          <div className="bill-links">
            <a href={bill.bill_url} target="_blank" rel="noopener noreferrer">
              View Bill
            </a>
            <a href={bill.explanatory_memo_url} target="_blank" rel="noopener noreferrer">
              Explanatory Memorandum
            </a>
          </div>
        </div>
      )}
    </div>
  );
}
```

#### 2.3 Create `BillsPanel.css` Styles
**Location**: `frontend/src/components/BillsPanel.css`

**Design Specifications**:
```css
.bills-panel {
  width: 320px;
  background: #ffffff;
  border-left: 1px solid #e0e0e0;
  display: flex;
  flex-direction: column;
  height: 100vh;
  box-shadow: -2px 0 8px rgba(0,0,0,0.05);
}

.bills-panel-header {
  padding: 16px;
  border-bottom: 1px solid #e0e0e0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.bills-panel-header h2 {
  font-size: 16px;
  margin: 0;
  color: #333;
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: #666;
  padding: 0;
  width: 24px;
  height: 24px;
  line-height: 24px;
}

.bills-filters {
  padding: 12px;
  border-bottom: 1px solid #e0e0e0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.bills-filters input,
.bills-filters select {
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 4px;
  font-size: 14px;
}

.bills-actions {
  padding: 12px;
  border-bottom: 1px solid #e0e0e0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.bills-actions button {
  padding: 6px 12px;
  background: #4a90e2;
  border: none;
  border-radius: 4px;
  color: white;
  cursor: pointer;
  font-size: 13px;
}

.bills-actions button:disabled {
  background: #ccc;
  cursor: not-allowed;
}

.last-updated {
  font-size: 11px;
  color: #999;
}

.bills-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.bill-card {
  background: #f9f9f9;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 8px;
  cursor: pointer;
  transition: box-shadow 0.2s;
}

.bill-card:hover {
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.bill-header {
  display: flex;
  justify-content: space-between;
  align-items: start;
  margin-bottom: 8px;
}

.bill-header h3 {
  font-size: 14px;
  margin: 0;
  color: #333;
  flex: 1;
  line-height: 1.4;
}

.expand-icon {
  font-size: 12px;
  color: #666;
  margin-left: 8px;
}

.bill-meta {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.chamber-badge,
.status-badge {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 500;
}

.chamber-badge {
  background: #e3f2fd;
  color: #1976d2;
}

.status-badge {
  background: #fff3e0;
  color: #f57c00;
}

.bill-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
}

.info-row {
  display: flex;
  justify-content: space-between;
}

.info-row .label {
  color: #666;
  font-weight: 500;
}

.bill-details {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #e0e0e0;
}

.bill-summary {
  font-size: 13px;
  line-height: 1.5;
  color: #555;
  margin: 0 0 12px 0;
}

.bill-links {
  display: flex;
  gap: 8px;
}

.bill-links a {
  padding: 6px 12px;
  background: #4a90e2;
  color: white;
  text-decoration: none;
  border-radius: 4px;
  font-size: 12px;
  transition: background 0.2s;
}

.bill-links a:hover {
  background: #357abd;
}

.loading,
.error,
.no-bills {
  padding: 24px;
  text-align: center;
  color: #666;
  font-size: 14px;
}

.error {
  color: #d32f2f;
}
```

#### 2.4 Update `api.js`
**Location**: `frontend/src/api.js`

**Add new method**:
```javascript
/**
 * Fetch all bills before parliament.
 * @param {object} options - Optional filters { refresh: bool, chamber: string }
 */
async getBills(options = {}) {
  const params = new URLSearchParams();
  if (options.refresh) params.append('refresh', 'true');
  if (options.chamber && options.chamber !== 'all') {
    params.append('chamber', options.chamber);
  }

  const url = `${API_BASE}/api/bills${params.toString() ? '?' + params.toString() : ''}`;
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error('Failed to fetch bills');
  }
  return response.json();
}
```

#### 2.5 Update `App.css`
**Location**: `frontend/src/App.css`

**Add responsive layout**:
```css
.app {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

/* Ensure main chat interface flexes properly */
.chat-interface {
  flex: 1;
  min-width: 0; /* Important for flex child with overflow */
}

/* Show/hide bills panel toggle button */
.toggle-bills-btn {
  position: fixed;
  top: 16px;
  right: 16px;
  padding: 8px 16px;
  background: #4a90e2;
  border: none;
  border-radius: 20px;
  color: white;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
  z-index: 1000;
  transition: all 0.2s;
}

.toggle-bills-btn:hover {
  background: #357abd;
  box-shadow: 0 3px 12px rgba(0,0,0,0.2);
}

/* Responsive: Hide bills panel on small screens */
@media (max-width: 1024px) {
  .bills-panel {
    position: fixed;
    right: 0;
    top: 0;
    z-index: 999;
  }
}
```

### Phase 3: Integration & Polish

#### 3.1 Add Toggle Button to UI
**Options**:
1. **In Sidebar Header** (Recommended):
   - Add button next to "New Conversation"
   - Icon: "📋" or "Bills" text

2. **In Main Chat Header**:
   - Floating button in top-right corner
   - More accessible, always visible

**Implementation** (Option 1 - Sidebar):
```jsx
// In Sidebar.jsx
<div className="sidebar-header">
  <h1>LLM Council</h1>
  <div className="header-buttons">
    <button className="new-conversation-btn" onClick={onNewConversation}>
      + New Conversation
    </button>
    <button className="toggle-bills-btn" onClick={onToggleBills}>
      📋 Bills
    </button>
  </div>
</div>
```

#### 3.2 Error Handling & Loading States
**Backend**:
- Graceful degradation on parse errors
- Timeout handling (30s per page)
- Retry logic for transient failures
- Detailed error logging

**Frontend**:
- Loading spinner during initial fetch
- Error message with retry button
- Empty state when no bills match filters
- Skeleton loaders for better UX

#### 3.3 Performance Optimizations
**Backend**:
- Implement caching (5-minute TTL)
- Rate limiting between page requests (500ms delay)
- Gzip compression on responses
- Connection pooling for HTTP requests

**Frontend**:
- Virtual scrolling for large bill lists (optional, if >100 bills)
- Debounced search input (300ms)
- Memoized filter functions
- Lazy load bill summaries (show on expand)

#### 3.4 Testing Strategy
**Backend Tests**:
1. Unit tests for HTML parsing
2. Mock HTTP tests for scraper
3. Integration tests for API endpoint
4. Cache behavior tests

**Frontend Tests**:
1. Component rendering tests
2. Filter functionality tests
3. API integration tests
4. Responsive layout tests

**Manual Testing**:
1. Verify all bill fields display correctly
2. Test pagination through multiple pages
3. Verify external links work
4. Test filter combinations
5. Test cache refresh functionality
6. Verify responsiveness on different screen sizes

### Phase 4: Optional Enhancements

#### 4.1 Background Auto-Refresh
- Goroutine refreshes cache every 30 minutes
- Broadcast updates via WebSocket (if implementing real-time)

#### 4.2 Bill Tracking/Favorites
- Add "star" functionality to save favorite bills
- Store in localStorage or backend database
- Notification when tracked bill status changes

#### 4.3 Advanced Filtering
- Date range filter (introduced in last X days)
- Multiple status filters
- Portfolio/sponsor filter dropdown

#### 4.4 Integration with Council
- "Ask Council About This Bill" button
- Pre-populates chat with bill context
- Council deliberates on bill implications

#### 4.5 Export Functionality
- Export filtered bills as CSV/JSON
- Generate summary report

## Technical Considerations

### Web Scraping Ethics & Legal
1. **Robots.txt**: Check `https://www.aph.gov.au/robots.txt` for crawl restrictions
2. **Rate Limiting**: Implement 500ms delays between requests
3. **User-Agent**: Set descriptive User-Agent header
4. **Caching**: Essential to minimize requests (5-minute TTL minimum)
5. **Terms of Service**: Review APH website terms

### Error Scenarios
1. **Website Structure Changes**:
   - Monitor for parse errors
   - Log failures for investigation
   - Implement version detection if possible

2. **Network Issues**:
   - Timeout after 30 seconds
   - Retry once with exponential backoff
   - Return cached data if available

3. **Partial Data**:
   - Continue with successfully parsed bills
   - Log missing fields
   - Display warnings in UI

### Scalability
- Current approach: Fetch all bills at once (likely <100 bills)
- If bill count grows significantly (>500):
  - Implement true pagination in API
  - Add database persistence
  - Consider scheduled background jobs

### Maintenance
- Website structure may change without notice
- Plan for regular monitoring and updates
- Consider adding health check endpoint
- Document HTML parsing logic thoroughly

## File Summary

### Backend Files to Create/Modify
1. ✅ `backend-go/scraper.go` - NEW (320+ LOC)
2. ✅ `backend-go/cache.go` - NEW (80+ LOC, optional)
3. ✅ `backend-go/models.go` - MODIFY (add Bill structs)
4. ✅ `backend-go/main.go` - MODIFY (add route, handler)
5. ✅ `backend-go/config.go` - MODIFY (add cache TTL config)

### Frontend Files to Create/Modify
1. ✅ `frontend/src/components/BillsPanel.jsx` - NEW (200+ LOC)
2. ✅ `frontend/src/components/BillsPanel.css` - NEW (250+ LOC)
3. ✅ `frontend/src/App.jsx` - MODIFY (add bills panel state, toggle)
4. ✅ `frontend/src/App.css` - MODIFY (3-column layout)
5. ✅ `frontend/src/api.js` - MODIFY (add getBills method)
6. ✅ `frontend/src/components/Sidebar.jsx` - MODIFY (add toggle button)
7. ✅ `frontend/src/components/Sidebar.css` - MODIFY (button styles)

### Dependencies to Add
```bash
# Backend
go get github.com/PuerkitoBio/goquery
go get golang.org/x/net/html
```

## Estimated Complexity
- **Backend Scraper**: Medium-High (HTML parsing, pagination logic)
- **Backend API**: Low (single endpoint, straightforward)
- **Frontend Component**: Medium (state management, filtering, styling)
- **Integration**: Low (clean separation of concerns)
- **Total LOC**: ~1,000 lines (600 backend, 400 frontend)

## Development Sequence
1. **Day 1**: Backend scraper + models + tests
2. **Day 2**: Backend API endpoint + caching
3. **Day 3**: Frontend BillsPanel component
4. **Day 4**: Integration + styling + polish
5. **Day 5**: Testing + bug fixes + documentation

## Success Criteria
- ✅ All bills from APH website display correctly
- ✅ Pagination automatically follows all pages
- ✅ Filtering works for chamber and search
- ✅ External links to ParlInfo work correctly
- ✅ Caching reduces load on APH servers
- ✅ UI is responsive and matches existing design
- ✅ Error states handled gracefully
- ✅ Loading states provide feedback
- ✅ No crashes on malformed data

## Risk Mitigation
1. **Website Structure Changes**:
   - Implement comprehensive error logging
   - Add health check endpoint that validates parsing

2. **Performance**:
   - Cache aggressively
   - Implement timeouts
   - Consider background refresh

3. **Data Quality**:
   - Validate all extracted fields
   - Log parsing issues
   - Gracefully handle missing data

4. **User Experience**:
   - Provide clear loading indicators
   - Show helpful error messages
   - Enable manual refresh

## Future Considerations
- Database persistence for historical bill tracking
- Change detection (notify when bill status updates)
- Bill comparison feature
- Integration with council deliberations
- Advanced search/filtering
- Export functionality
- Mobile-optimized view
