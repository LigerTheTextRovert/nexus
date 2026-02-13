// Package proxy
package proxy

import (
	"io"
	"log"
	"net/http"

	"github.com/LigerTheTextRovert/nexus/internal/config"
)

func ProxyHandler(w http.ResponseWriter, r *http.Request, c *config.Config) {
	proxyRequest, _ := http.NewRequest(r.Method, c.BackendURL+r.URL.Path, r.Body)

	proxyRequest.Header = r.Header.Clone()

	log.Printf("[PROXY] Frowarding to:%s", c.BackendURL+r.URL.Path)

	client := &http.Client{}
	res, err := client.Do(proxyRequest)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("[ERROR] An error occur: %s", err.Error())
		return
	}
	log.Printf("[RESPONSE] Backend: %d, %s", res.StatusCode, http.StatusText(res.StatusCode))

	for key, values := range res.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(res.StatusCode)
	log.Printf("[RESPONSE] Client: %d, %s", res.StatusCode, http.StatusText(res.StatusCode))

	io.Copy(w, res.Body)
	defer res.Body.Close()
}
