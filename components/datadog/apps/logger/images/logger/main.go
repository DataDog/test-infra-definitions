package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	handler := http.HandlerFunc(handleRequest)
	http.Handle("/", handler)

	err := http.ListenAndServe(":3333", nil)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
		os.Exit(1)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.Copy(os.Stdout, r.Body)
	fmt.Println()
}
