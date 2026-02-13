// Package proxy
package proxy

import (
	"io"
	"net/http"

	"github.com/LigerTheTextRovert/nexus/internal/config"
)

func ProxyHandler(w http.ResponseWriter, r *http.Request, c *config.Config) {
	proxyRequest, _ := http.NewRequest(r.Method, c.BackendURL+r.URL.Path, r.Body)

	proxyRequest.Header = r.Header.Clone()

	client := &http.Client{}
	res, err := client.Do(proxyRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for key, values := range res.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(res.StatusCode)

	io.Copy(w, res.Body)
	defer res.Body.Close()
}
