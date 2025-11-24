package main

import (
	"sync"
	"time"
)

// BillsCache provides thread-safe caching for bills data
type BillsCache struct {
	mu          sync.RWMutex
	bills       []Bill
	lastUpdated time.Time
	ttl         time.Duration
}

// NewBillsCache creates a new bills cache with the specified TTL
func NewBillsCache(ttl time.Duration) *BillsCache {
	return &BillsCache{
		ttl: ttl,
	}
}

// Get retrieves bills from cache if not expired
// Returns the bills and a boolean indicating if the cache hit was successful
func (c *BillsCache) Get() ([]Bill, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if cache is empty
	if len(c.bills) == 0 {
		return nil, false
	}

	// Check if cache has expired
	if time.Since(c.lastUpdated) > c.ttl {
		return nil, false
	}

	// Return cached bills (make a copy to prevent external modifications)
	billsCopy := make([]Bill, len(c.bills))
	copy(billsCopy, c.bills)

	return billsCopy, true
}

// Set updates the cache with new bills data
func (c *BillsCache) Set(bills []Bill) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store bills (make a copy to prevent external modifications)
	c.bills = make([]Bill, len(bills))
	copy(c.bills, bills)
	c.lastUpdated = time.Now()
}

// Clear removes all bills from the cache
func (c *BillsCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.bills = nil
	c.lastUpdated = time.Time{}
}

// GetLastUpdated returns when the cache was last updated
func (c *BillsCache) GetLastUpdated() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lastUpdated
}

// IsExpired checks if the cache has expired
func (c *BillsCache) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.bills) == 0 {
		return true
	}

	return time.Since(c.lastUpdated) > c.ttl
}

// GetSize returns the number of bills in the cache
func (c *BillsCache) GetSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.bills)
}
