package main

import (
	"fmt"
	"strings"

	"golang.org/x/text/cases"
)

const (
	AppName = "Vein Framework"
	Version = "0.1"
)

type Response struct {
	Greet string
}

func main() {
	fmt.Println(RUN(nil))
}


func RUN(input interface{}) *Response {
	response := &Response{}
	switch v := input.(type){
		case string:
			response.Greet = strings.ToLower(v)
		case nil:
			response.Greet = strings.ToLower(Ap)


	}
	return &Response{
		Greet: strings.ToLower(AppName),
	}
}