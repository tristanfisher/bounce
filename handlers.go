package main

import (
	"bounce/request"
	"github.com/rs/zerolog"
	"net/http"
)

type HandlerSupport struct {
	Log       zerolog.Logger
	HttpAddr  string
	HttpsAddr string
}

func Mux(hs HandlerSupport) *http.ServeMux {
	// our default route for this is to detect if we are receiving requests for the target domain.
	// if so, we redirect to the requested destination
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ri := request.GetRequestInfo(r)
		if ri.RequestedHost == hs.HttpAddr || ri.RequestedHost == hs.HttpsAddr {
			http.Redirect(w, r, "/", http.StatusFound)
		}
		http.NotFound(w, r)
	})
	return mux
}
