package logging

import (
	"log"
	"net/http"
	"time"
)

func LoggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		log.Printf("[REQUEST] %s  %s", r.Method, r.URL.Path)

		for key, values := range r.Header {
			for _, value := range values {
				log.Printf("[HEADER] %s : %s", key, value)
			}
		}

		h.ServeHTTP(w, r)

		duration := time.Since(startTime)

		log.Printf("[DURATION] Request took %v", duration)
	})
}
