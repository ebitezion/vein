package data

import (
	"testing"

	"github.com/ebitezion/vein/internal/validator"
)

func TestValidateFiltersValid(t *testing.T) {
	v := validator.New()
	f := Filters{
		Page:         1,
		PageSize:     20,
		Sort:         "created_at",
		SortSafelist: []string{"created_at", "-created_at", "email", "-email"},
	}

	ValidateFilters(v, f)

	if !v.Valid() {
		t.Fatalf("expected filters to be valid, got errors: %+v", v.Errors)
	}
}

func TestValidateFiltersInvalidPage(t *testing.T) {
	v := validator.New()
	f := Filters{
		Page:         0,
		PageSize:     20,
		Sort:         "created_at",
		SortSafelist: []string{"created_at", "-created_at"},
	}

	ValidateFilters(v, f)

	if _, ok := v.Errors["page"]; !ok {
		t.Fatal("expected page validation error")
	}
}

func TestValidateFiltersInvalidPageSize(t *testing.T) {
	v := validator.New()
	f := Filters{
		Page:         1,
		PageSize:     101,
		Sort:         "created_at",
		SortSafelist: []string{"created_at", "-created_at"},
	}

	ValidateFilters(v, f)

	if _, ok := v.Errors["page_size"]; !ok {
		t.Fatal("expected page_size validation error")
	}
}

func TestValidateFiltersInvalidSort(t *testing.T) {
	v := validator.New()
	f := Filters{
		Page:         1,
		PageSize:     20,
		Sort:         "bad",
		SortSafelist: []string{"created_at", "-created_at"},
	}

	ValidateFilters(v, f)

	if got, ok := v.Errors["sort"]; !ok || got != "invalid sort value" {
		t.Fatalf("expected sort validation error, got: %+v", v.Errors)
	}
}

func TestFiltersSortColumn(t *testing.T) {
	f := Filters{
		Sort:         "-created_at",
		SortSafelist: []string{"created_at", "-created_at"},
	}

	if got := f.sortColumn(); got != "created_at" {
		t.Fatalf("expected sort column created_at, got %s", got)
	}
}

func TestFiltersSortDirection(t *testing.T) {
	asc := Filters{Sort: "created_at"}
	if got := asc.sortDirection(); got != "ASC" {
		t.Fatalf("expected ASC, got %s", got)
	}

	desc := Filters{Sort: "-created_at"}
	if got := desc.sortDirection(); got != "DESC" {
		t.Fatalf("expected DESC, got %s", got)
	}
}

func TestFiltersSortColumnPanicsForUnsafeSort(t *testing.T) {
	f := Filters{
		Sort:         "unsafe",
		SortSafelist: []string{"created_at", "-created_at"},
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unsafe sort parameter")
		}
	}()

	_ = f.sortColumn()
}
