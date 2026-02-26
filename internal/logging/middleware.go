// Package logging
package logging

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

type StatusLog struct {
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	UserAgent  string `json:"user_agent"`
	RemoteIP   string `json:"remote_ip"`
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		sw := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(sw, r)
		if sw.status == 0 {
			sw.status = http.StatusOK
		}

		request := &StatusLog{
			Method:     r.Method,
			Path:       r.URL.Path,
			Status:     sw.status,
			DurationMS: time.Since(startTime).Microseconds(),
			RemoteIP:   r.RemoteAddr,
			UserAgent:  r.UserAgent(),
		}

		bytes, _ := json.Marshal(request)
		log.Println(string(bytes))

	})
}
