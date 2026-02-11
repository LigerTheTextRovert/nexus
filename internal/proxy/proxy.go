// Package proxy
package proxy

import (
	"io"
	"net/http"
)

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	pRes, _ := http.NewRequest(r.Method, "http://localhost:3001"+r.URL.Path, r.Body)

	pRes.Header = r.Header.Clone()

	client := &http.Client{}
	res, err := client.Do(pRes)
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
