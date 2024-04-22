package data

import (
	"strings"

	"github.com/michaelgov-ctrl/memebase/internal/validator"
)

type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafelist []string
}

func ValidateFilters(v *validator.Validator, f Filters) {
	v.Check(f.Page > 0, "page", "must be greater than 0")
	v.Check(f.Page <= 1_000, "page", "must be a maximum of 1 thousand")
	v.Check(f.PageSize > 0, "page_size", "must be greater than 0")
	v.Check(f.PageSize <= 10, "page_size", "must be a maximum of 10")

	v.Check(validator.PermittedValue(f.Sort, f.SortSafelist...), "sort", "invalid sort value")
}

func (f Filters) sortField() string {
	for _, safeValue := range f.SortSafelist {
		if f.Sort == safeValue {
			return strings.TrimPrefix(f.Sort, "-")
		}
	}

	panic("unsafe sort parameter: " + f.Sort)
}

func (f Filters) sortDirection() int {
	if strings.HasPrefix(f.Sort, "-") {
		return -1
	}

	return 1
}

func (f Filters) limit() int {
	return f.PageSize
}

func (f Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}

type Metadata struct {
	CurrentPage  int `json:"current_page,omitempty"`
	PageSize     int `json:"page_size,omitempty"`
	FirstPage    int `json:"first_page,omitempty"`
	LastPage     int `json:"last_page,omitempty"`
	TotalRecords int `json:"total_records,omitempty"`
}

func (m *Metadata) Calculate(filteredRecordCount, totalRecords, page, pageSize int) {
	if filteredRecordCount == 0 {
		return
	}

	m.CurrentPage = page
	m.PageSize = pageSize
	m.FirstPage = 1
	m.LastPage = (totalRecords + pageSize - 1) / pageSize
	m.TotalRecords = totalRecords
}
