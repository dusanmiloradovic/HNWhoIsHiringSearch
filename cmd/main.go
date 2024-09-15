package main

import (
	"HNtest/internal/hnfetch"
	"log"
	"net/http"
)

func main() {
	hnFetch := hnfetch.NewHNFetch()
	err := hnFetch.Init()
	if err != nil {
		panic("Not able to start the server")
	}
	latestJobsHandler := func(w http.ResponseWriter, r *http.Request) {
		hnFetch.GetPosts()
	}
	http.HandleFunc("/latest-jobs", latestJobsHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
