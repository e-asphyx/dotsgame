package main

import (
	"log"
	"net/http"
)

func main() {
	/* Serve static */
	http.Handle("/static/", http.FileServer(http.Dir("")))

	/* Start server */
	log.Fatal(http.ListenAndServe(":8080", nil))
}
