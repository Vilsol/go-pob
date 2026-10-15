//go:build !noimportproxy

package main

import (
	"net/http"
	"time"

	"github.com/Vilsol/go-pob/internal/importproxy"
)

func apiHandler() http.Handler {
	return importproxy.Handler(importproxy.Deps{
		Client: &http.Client{Timeout: 30 * time.Second},
		Now:    time.Now,
		Cache:  importproxy.NewMemoryCache(),
	})
}
