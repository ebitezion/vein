package main

import "fmt"

const AppName ="VEIN BACKEND FRAM"
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
