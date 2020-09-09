package common

import (
	"context"
	"fmt"
	"net/http"
)

type OriginalRequestURIContextKey string

const RequestURIContext OriginalRequestURIContextKey = "OriginalRequestURI"

// HTTP middleware setting original request URL on context
func SetOriginalRequestURI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var uri string
		if len(r.Header.Get("X-Forwarded-For")) > 0 {
			uri = r.Header.Get("X-Forwarded-For")
		} else {
			if r.URL.IsAbs() && len(r.URL.Opaque) == 0 {
				uri = r.URL.String()
			} else {
				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}
				uri = fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)
			}
		}
		ctx := context.WithValue(r.Context(), RequestURIContext, uri)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
