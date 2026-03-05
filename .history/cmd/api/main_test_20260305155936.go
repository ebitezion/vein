package main

import "testing"

func TestMain(t *testing.T) {
  input:= Response{
	Greet: AppName,
  }
  expected := strings.toinput.Greet

  got:= RUN(AppName).Greet

  if expected != got{
   t.Errorf("Expected %s Got %s",expected,got)
  }
}
