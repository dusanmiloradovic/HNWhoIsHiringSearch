package main

import (
	"HNtest/internal/hnfetch"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"strconv"
)

func main() {
	hnFetch := hnfetch.NewHNFetch()
	err := hnFetch.Init()
	if err != nil {
		panic("Not able to start the server")
	}
	latestJobsHandler := func(w http.ResponseWriter, r *http.Request) {
		fetchSize, err := strconv.Atoi(r.PathValue("fetchSize"))
		if err != nil {
			slog.Error("Error parsing the path", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		cursor := r.PathValue("cursor")
		pCursor := -1
		if cursor != "start" {
			pCursor, err = strconv.Atoi(cursor)
			if err != nil {
				slog.Error("Error parsing the path", err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}
		}
		ret := hnFetch.GetPosts(pCursor, fetchSize)
		jData, err := json.Marshal(ret)
		if err != nil {
			slog.Error("Error marshalling data", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jData)
	}
	http.HandleFunc("/latest-jobs/{fetchSize}/{cursor}", latestJobsHandler)
	go hnFetch.BackgroundCheck()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
