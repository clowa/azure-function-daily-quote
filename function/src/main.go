package main

import (
	"log"
	"net/http"

	"github.com/clowa/az-func-daily-quote/src/handlers"
	"github.com/clowa/az-func-daily-quote/src/lib/config"
)

func main() {
	globalConfig := config.GetConfig()
	globalConfig.LoadConfig()

	listenAddr := globalConfig.ApiPort
	http.HandleFunc("/api/quote", handlers.QuoteHandler)
	log.Printf("About to listen on %s. Go to https://127.0.0.1%s/", listenAddr, listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
