// Package gateway
package gateway

import (
	"fmt"
	"log"
	"net/http"

	"github.com/LigerTheTextRovert/nexus/internal/config"
	"github.com/LigerTheTextRovert/nexus/internal/proxy"
)

func main() {

	var cfg config.Config
	_, err := config.LoadConfig("../../configs/config.yaml", &cfg)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "gateway is running...")
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		proxy.ProxyHandler(w, r, &cfg)
	})

	port := cfg.Port
	log.Printf("Starting gateway on port %s...", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
