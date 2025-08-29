package model

// Page represents a generic paginated response
type Page[T any] struct {
	Content          []T   `json:"content"`
	Number           int   `json:"number"`
	Size             int   `json:"size"`
	TotalElements    int64 `json:"totalElements"`
	TotalPages       int   `json:"totalPages"`
	NumberOfElements int   `json:"numberOfElements"`
}

// NewPage creates a new Page instance with calculated values
func NewPage[T any](content []T, number int, size int, totalElements int64) *Page[T] {
	totalPages := int((totalElements + int64(size) - 1) / int64(size))
	if totalElements == 0 {
		totalPages = 0
	}

	return &Page[T]{
		Content:          content,
		Number:           number,
		Size:             size,
		TotalElements:    totalElements,
		TotalPages:       totalPages,
		NumberOfElements: len(content),
	}
}
