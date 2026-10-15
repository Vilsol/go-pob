package host

import (
	"net/http"
	"strings"
)

func Routes(api, native http.Handler) func(http.Handler) http.Handler {
	nativeStripped := http.StripPrefix("/desktop", native)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasPrefix(r.URL.Path, "/api/"):
				api.ServeHTTP(w, r)
			case strings.HasPrefix(r.URL.Path, "/desktop/"):
				nativeStripped.ServeHTTP(w, r)
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}
