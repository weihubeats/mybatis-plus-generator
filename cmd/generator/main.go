package main

import (
	"fmt"
	"log"
	"mybatis-plus-generator/internal/handler"
	"net/http"
)

func main() {
	http.HandleFunc("/", handler.GenerateHandler)

	address := ":8080"
	fmt.Printf("Server is running on http://localhost%s\n", address)
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
