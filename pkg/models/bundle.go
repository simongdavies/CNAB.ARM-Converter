package models

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/go-chi/render"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/helpers"
)

// BundleContextKey is the type used for the keys of items placed in the request context
type BundleContextKey string

const (
	TemplateGeneratorPath       string           = "/api/generate/template"
	NestedResourceGeneratorPath string           = "/api/generate/deployment"
	BundleContext               BundleContextKey = "bundle"
)

type Bundle struct {
	Ref              string
	Force            bool
	InsecureRegistry bool
	Simplyfy         bool
}

func BundleCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// get the image name

		imageName := strings.TrimPrefix(r.URL.Path, TemplateGeneratorPath)
		imageName = strings.TrimPrefix(imageName, NestedResourceGeneratorPath)
		imageName = strings.TrimPrefix(imageName, "/")

		if len(imageName) == 0 {
			_ = render.Render(w, r, helpers.ErrorInvalidRequest("Image Name missing in path"))
			return
		}

		imageName = strings.TrimPrefix(imageName, "/")
		_, err := reference.ParseAnyReference(imageName)
		if err != nil {
			_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to parse image reference: %s error: %v", imageName, err)))
			return
		}
		bundleContext := Bundle{
			Ref:              imageName,
			Force:            getQueryParam(r, "force"),
			InsecureRegistry: getQueryParam(r, "insecureregistry"),
			Simplyfy:         getQueryParam(r, "simplyfy"),
		}

		ctx := context.WithValue(r.Context(), BundleContext, &bundleContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getQueryParam(r *http.Request, name string) bool {
	result := false
	val := r.URL.Query().Get(name)
	if len(val) > 0 && strings.ToLower(val) == "true" {
		result = true
	}

	return result
}
