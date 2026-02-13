package logging

import (
	"log"
	"net/http"
	"time"
)

func LoggerMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		h.ServeHTTP(w, r)
		duration := time.Since(startTime)
		log.Printf("Request took %vs", duration)
	})
}
