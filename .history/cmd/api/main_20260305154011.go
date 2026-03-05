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
	fmt.Println(RUN(nil))
}


func RUN(input interface{}) *Response {

	switch v := input.(type){
		
	}
	return &Response{
		Greet: strings.ToLower(AppName),
	}
}