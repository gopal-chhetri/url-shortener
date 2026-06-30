package utils

import (
	"math"
	"strconv"
)

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Page        int    `json:"page" query:"page"`
	PageSize    int    `json:"page_size" query:"page_size"`
	SearchParam string `json:"search_param" query:"search_param"`
}

// PaginationResponse represents pagination metadata in responses
type PaginationResponse struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// NewPaginationResponse creates a pagination response
func NewPaginationResponse(total int64, page, pageSize int) *PaginationResponse {
	if pageSize <= 0 {
		pageSize = 10
	}
	if page <= 0 {
		page = 1
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &PaginationResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// ParsePaginationParams parses pagination parameters from query strings
func ParsePaginationParams(pageStr, pageSizeStr string, defaultPage, defaultPageSize int) (page, pageSize int) {
	page = defaultPage
	pageSize = defaultPageSize

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	return page, pageSize
}

// GetOffset calculates the offset for database queries
func (p PaginationRequest) GetOffset() int32 {
	return int32((p.Page - 1) * p.PageSize)
}

// GetLimit returns the limit for database queries
func (p PaginationRequest) GetLimit() int32 {
	return int32(p.PageSize)
}
