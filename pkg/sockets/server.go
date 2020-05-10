package sockets

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Debug().Interface("rurl", r.URL).Msg("")
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}
