package main

import (
	"strings"
	"testing"
)

func TestMain(t *testing.T) {
  input:= Response{
	Greet: AppName,
  }
  expected := strings.ToLower(input.Greet)

  got:= RUN(AppName).Greet

  if expected != got{
   t.Errorf("Expected %s Got %s",expected,got)
  }
}
