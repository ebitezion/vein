package main

import (
	"fmt"
	"strings"
)

const (
	AppName = "Vein Framework"
	Version = "0.1"
)

type Response struct {
	Greet string
}

func main() {
	fmt.Println("Hello Vein")
}

func RUN(input interface{}) *Response {

	var response *Response

	response.Greet = strings.ToLower(AppName)

	return response
}
