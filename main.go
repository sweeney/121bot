package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	http.HandleFunc("/", rootHandler)
	http.ListenAndServe(":"+port, nil)

}

func rootHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintln(w, "hello world")
	return

}
