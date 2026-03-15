package validator

import (
	"regexp"
	"testing"
)

func TestValidatorCheckAndValid(t *testing.T) {
	v := New()

	v.Check(false, "field", "must not be empty")
	if v.Valid() {
		t.Fatalf("expected validator to be invalid")
	}
	if got := v.Errors["field"]; got != "must not be empty" {
		t.Fatalf("unexpected error message: %s", got)
	}
}

func TestIn(t *testing.T) {
	if !In("admin", "user", "admin") {
		t.Fatalf("expected value to be found")
	}
	if In("guest", "user", "admin") {
		t.Fatalf("did not expect value to be found")
	}
}

func TestMatches(t *testing.T) {
	rx := regexp.MustCompile(`^test-[0-9]+$`)
	if !Matches("test-123", rx) {
		t.Fatalf("expected string to match regex")
	}
	if Matches("bad", rx) {
		t.Fatalf("expected string to not match regex")
	}
}

func TestUnique(t *testing.T) {
	values := []string{"a", "b", "c"}
	if !Unique(values) {
		t.Fatalf("expected slice to be unique")
	}

	values = []string{"a", "b", "a"}
	if Unique(values) {
		t.Fatalf("expected slice to be non-unique")
	}
}
