package main

import (
	"GO-Proxie/services"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/dns-query", services.DoHandler)
	log.Println("Starting proxy on :8087")
	log.Fatal(http.ListenAndServe(":8087", nil))
}
