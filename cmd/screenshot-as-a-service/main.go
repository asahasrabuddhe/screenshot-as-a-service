package main

import (
	"go.ajitem.com/screenshot"
	"log"
	"net/http"
)

func main() {
	http.Handle("/", screenshot.NewScreenshot("/Users/ajitem/Downloads/chrome-mac/Chromium.app/Contents/MacOS/Chromium", 12345))
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal(err)
	}
}
