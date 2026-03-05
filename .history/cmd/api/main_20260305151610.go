package main

import "fmt"

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

func RUN() *Response {

	var response *Response

	response.Greet = AppName.to

	return
}
