package common

import (
	"context"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type OriginalRequestURIContextKey string

const RequestURIContext OriginalRequestURIContextKey = "OriginalRequestURI"

// HTTP middleware setting original request URL on context
func SetOriginalRequestURI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var uri string
		if r.URL.IsAbs() && len(r.URL.Opaque) == 0 {
			uri = r.URL.String()
		} else {
			uri = fmt.Sprintf("https://%s%s", r.Host, r.RequestURI)
		}
		log.Infof("Request URI: %s", uri)
		ctx := context.WithValue(r.Context(), RequestURIContext, uri)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
