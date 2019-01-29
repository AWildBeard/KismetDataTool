package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("Hello World!")
	if response, err := http.Get("https://www.google.com") ; err == nil {
		fmt.Println("Sucessfully connected to www.google.com.")
		fmt.Printf("Response: %v\n", response)
		fmt.Printf("%T ", response.Body)
	}
}
