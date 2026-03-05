package main

import "fmt"

const AppName ="Vein Framework"
type Response struct {
	Greet string
}

func main() {
	fmt.Println("Hello Vein")
}


func RUN() *Response{

	var response *Response;

	response.Greet= "Hello Welcome "


	return 
}
