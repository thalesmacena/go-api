package main

import (
	"go-api/pkg/log"
	"math"
	"sync"
	"time"
)

// ResourceCache Simple cache to store the totals of the resources
type ResourceCache struct {
	mu    sync.RWMutex
	cache map[string]int
}

// NewResourceCache creates a new ResourceCache
func NewResourceCache() *ResourceCache {
	return &ResourceCache{
		cache: make(map[string]int),
	}
}

// Get retrieves a value from the cache
func (c *ResourceCache) Get(key string) (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists := c.cache[key]
	return value, exists
}

// Set stores a value in the cache
func (c *ResourceCache) Set(key string, value int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = value
}

// ResourceCache global for the resources
var resourceCache = NewResourceCache()

// PageResponse represents the standardized response of a paginated search
type PageResponse[T any] struct {
	Items         []T
	NumberOfItems int   // numberOfElements
	PageSize      int   // pageSize
	TotalElements int64 // totalElements
	TotalPages    int   // totalPages
}

// MockResource simulates a paginated resource with immutable data in memory
type MockResource[T any] struct {
	data []T
}

// NewMockResource creates a new MockResource
func NewMockResource[T any](data []T) *MockResource[T] {
	return &MockResource[T]{data: data}
}

// Count simulates a dedicated endpoint that returns the total number of elements
func (r *MockResource[T]) Count() int64 {
	// Simula latência do endpoint de count
	time.Sleep(100 * time.Millisecond)
	return int64(len(r.data))
}

// FetchPage returns up to size items of the specified page (offset = page * size)
func (r *MockResource[T]) FetchPage(page int, size int) PageResponse[T] {
	// Simula latência da consulta de página
	time.Sleep(100 * time.Millisecond)

	if size <= 0 {
		return PageResponse[T]{
			Items:         []T{},
			NumberOfItems: 0,
			PageSize:      size,
			TotalElements: int64(len(r.data)),
			TotalPages:    0,
		}
	}

	totalElements := int64(len(r.data))
	totalPages := int(math.Ceil(float64(totalElements) / float64(size)))

	if page < 0 || page >= totalPages {
		return PageResponse[T]{
			Items:         []T{},
			NumberOfItems: 0,
			PageSize:      size,
			TotalElements: totalElements,
			TotalPages:    totalPages,
		}
	}

	offset := page * size
	end := offset + size
	if end > len(r.data) {
		end = len(r.data)
	}

	items := append([]T(nil), r.data[offset:end]...)
	return PageResponse[T]{
		Items:         items,
		NumberOfItems: len(items),
		PageSize:      size,
		TotalElements: totalElements,
		TotalPages:    totalPages,
	}
}

// CombinedPaginatedSearch combines the pagination of A (prioritized) followed by B
func CombinedPaginatedSearch[T any](
	resourceA *MockResource[T],
	resourceB *MockResource[T],
	page int,
	size int,
) PageResponse[T] {
	if size <= 0 {
		return PageResponse[T]{Items: []T{}}
	}

	var totalA, totalB int
	var wgTotals sync.WaitGroup

	// Check cache for resource A
	if cachedA, exists := resourceCache.Get("resourceA"); exists {
		totalA = cachedA
	} else {
		wgTotals.Add(1)
		go func() {
			defer wgTotals.Done()
			log.Infof("Fetching count from resource %s", "A")
			totalA = int(resourceA.Count())
			resourceCache.Set("resourceA", totalA)
		}()
	}

	// Check cache for resource B
	if cachedB, exists := resourceCache.Get("resourceB"); exists {
		totalB = cachedB
	} else {
		wgTotals.Add(1)
		go func() {
			defer wgTotals.Done()
			log.Infof("Fetching count from resource %s", "B")
			totalB = int(resourceB.Count())
			resourceCache.Set("resourceB", totalB)
		}()
	}

	wgTotals.Wait()

	total := totalA + totalB

	if total == 0 {
		return PageResponse[T]{
			Items:         []T{},
			NumberOfItems: 0,
			PageSize:      size,
			TotalElements: 0,
			TotalPages:    0,
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(size)))
	if page < 0 || page >= totalPages {
		return PageResponse[T]{
			Items:         []T{},
			NumberOfItems: 0,
			PageSize:      size,
			TotalElements: int64(total),
			TotalPages:    totalPages,
		}
	}

	start := page * size
	var merged []T

	// Calculate how many items we need from A and B
	var aItems, bItems int

	if start < totalA {
		// How many items can we get from A
		aItems = totalA - start
		if aItems > size {
			aItems = size
		}
	}

	// Calculate how many items we still need (limited by the total available)
	remainingItems := size - aItems
	totalRemaining := total - start - aItems

	bItems = remainingItems
	if bItems > totalRemaining {
		bItems = totalRemaining
	}
	if bItems < 0 {
		bItems = 0
	}

	// Determine if we need to search for resource A
	aNeeded := aItems > 0
	var aPage, aSize int

	if aNeeded {
		aPage = start / size
		aSize = size // always get a complete page for simplicity
	}

	// Determine if we need to search for resource B
	bNeeded := bItems > 0
	var bPage, bSize int

	if bNeeded {
		bStart := start - totalA
		if bStart < 0 {
			bStart = 0
		}
		bPage = bStart / size
		bSkip := bStart % size

		// Need to get enough items considering the skip
		bSize = bItems + bSkip
		if bSize < size {
			bSize = size // at least one complete page
		}
	}

	var respA PageResponse[T]
	var respB PageResponse[T]

	if aNeeded && bNeeded {
		var wg sync.WaitGroup
		wg.Add(2)
		// Fetch A em paralelo
		go func(p, s int) {
			defer wg.Done()
			log.Infof("Fetching page from resource %s: page=%d size=%d", "A", p, s)
			respA = resourceA.FetchPage(p, s)
		}(aPage, aSize)
		// Fetch B em paralelo
		go func(p, s int) {
			defer wg.Done()
			log.Infof("Fetching page from resource %s: page=%d size=%d", "B", p, s)
			respB = resourceB.FetchPage(p, s)
		}(bPage, bSize)
		wg.Wait()
	} else if aNeeded {
		log.Infof("Fetching page from resource %s: page=%d size=%d", "A", aPage, aSize)
		respA = resourceA.FetchPage(aPage, aSize)
	} else {
		log.Infof("Fetching page from resource %s: page=%d size=%d", "B", bPage, bSize)
		respB = resourceB.FetchPage(bPage, bSize)
	}

	// Order: A then B
	if aNeeded && len(respA.Items) > 0 {
		// Calculate skip and how many items to get from A
		aSkip := start % size
		itemsToTake := respA.Items

		// Apply skip if necessary
		if aSkip > 0 && aSkip < len(itemsToTake) {
			itemsToTake = itemsToTake[aSkip:]
		}

		// Limit to what we really need from A
		if len(itemsToTake) > aItems {
			itemsToTake = itemsToTake[:aItems]
		}

		merged = append(merged, itemsToTake...)
	}

	if bNeeded && len(respB.Items) > 0 {
		// Calculate skip and how many items to get from B
		bStart := start - totalA
		if bStart < 0 {
			bStart = 0
		}
		bSkip := bStart % size

		itemsToTake := respB.Items

		// Apply skip if necessary
		if bSkip > 0 && bSkip < len(itemsToTake) {
			itemsToTake = itemsToTake[bSkip:]
		}

		// Limit to what we still need from B
		if len(itemsToTake) > bItems {
			itemsToTake = itemsToTake[:bItems]
		}

		merged = append(merged, itemsToTake...)
	}

	return PageResponse[T]{
		Items:         merged,
		NumberOfItems: len(merged),
		PageSize:      size,
		TotalElements: int64(total),
		TotalPages:    totalPages,
	}
}

func main() {
	resourceA := NewMockResource([]int{1, 2, 3, 4, 5, 6, 7, 8})
	resourceB := NewMockResource([]int{9, 10, 11, 12, 13})
	size := 3
	lastPage := 1

	// Query for pages 0.N dynamically, using lastPage from the return
	for page := 0; page < lastPage; page++ {
		resp := CombinedPaginatedSearch[int](resourceA, resourceB, page, size)
		lastPage = resp.TotalPages
		log.Infof("page=%d size=%d => items=%v numberOfElements=%d pageSize=%d totalElements=%d totalPages=%d",
			page, size, resp.Items, resp.NumberOfItems, resp.PageSize, resp.TotalElements, resp.TotalPages,
		)
	}
}
