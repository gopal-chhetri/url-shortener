package utils

import (
	"math"
	"strconv"

	"github.com/labstack/echo/v4"
)

type PaginationRequest struct {
	Page        int    `json:"page" query:"page"`
	PageSize    int    `json:"page_size" query:"page_size"`
	SearchParam string `json:"search_param" query:"search_param"`
}

type PaginationResponse struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

func NewPaginationRequest(c echo.Context) PaginationRequest {
	page := 1
	pageSize := 10

	if p := c.QueryParam("page"); p != "" {
		if parsedPage, err := strconv.Atoi(p); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	if ps := c.QueryParam("page_size"); ps != "" {
		if parsedPageSize, err := strconv.Atoi(ps); err == nil && parsedPageSize > 0 && parsedPageSize <= 100 {
			pageSize = parsedPageSize
		}
	}

	return PaginationRequest{
		Page:        page,
		PageSize:    pageSize,
		SearchParam: c.QueryParam("search_param"),
	}
}

func (p PaginationRequest) GetOffset() int32 {
	return int32((p.Page - 1) * p.PageSize)
}

func (p PaginationRequest) GetLimit() int32 {
	return int32(p.PageSize)
}

func NewPaginationResponse(req PaginationRequest, total int64, data interface{}) PaginationResponse {
	totalPages := int(math.Ceil(float64(total) / float64(req.PageSize)))

	return PaginationResponse{
		Page:       req.Page,
		PageSize:   req.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}
