//go:build noimportproxy

package main

import "net/http"

func apiHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte("this build of go-pob has no import proxy (built with -tags noimportproxy)"))
	})
}
