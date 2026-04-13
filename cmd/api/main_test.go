package main

import (
	"strings"
	"testing"
)

func TestMain(t *testing.T) {
	input := response{
		Greet: AppName,
	}
	expected := strings.ToLower(input.Greet)

	got := run(AppName).Greet

	if expected != got {
		t.Errorf("Expected %s Got %s", expected, got)
	}
}
