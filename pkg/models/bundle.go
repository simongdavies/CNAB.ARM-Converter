package models

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/go-chi/render"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/helpers"
	log "github.com/sirupsen/logrus"
)

// BundleContextKey is the type used for the keys of items placed in the request context
type BundleContextKey string

const (
	TemplateGeneratorPath       string           = "/api/generate/template"
	NestedResourceGeneratorPath string           = "/api/generate/deployment"
	RedirectPath                string           = "/api/redirect"
	UIRedirectPath              string           = "/api/customui"
	UIDefPath                   string           = "/api/generate/uidef"
	BundleContext               BundleContextKey = "bundle"
)

type Bundle struct {
	Ref               string
	Force             bool
	InsecureRegistry  bool
	Simplyfy          bool
	Timeout           int
	ReplaceKubeconfig bool
}

func BundleCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// get the image name

		imageName := strings.TrimPrefix(r.URL.Path, TemplateGeneratorPath)
		imageName = strings.TrimPrefix(imageName, NestedResourceGeneratorPath)
		imageName = strings.TrimPrefix(imageName, UIDefPath)
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
			Ref:               imageName,
			Force:             getBoolQueryParam(r, "force"),
			InsecureRegistry:  getBoolQueryParam(r, "insecureregistry"),
			Simplyfy:          getBoolQueryParam(r, "simplyfy"),
			Timeout:           getIntQueryParam(r, "timeout", 15),
			ReplaceKubeconfig: getBoolQueryParam(r, "useaks"),
		}

		ctx := context.WithValue(r.Context(), BundleContext, &bundleContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getBoolQueryParam(r *http.Request, name string) bool {
	result := false
	for k, v := range r.URL.Query() {
		// ignore multiple values
		if strings.EqualFold(k, name) && (len(v[0]) == 0 || strings.ToLower(v[0]) == "true") {
			result = true
			break
		}
	}
	return result
}

func getIntQueryParam(r *http.Request, name string, defaultValue int) int {
	result := defaultValue
	for k, v := range r.URL.Query() {
		// ignore multiple values
		if strings.EqualFold(k, name) && (len(v[0]) > 0) {
			if val, err := strconv.Atoi(v[0]); err == nil {
				if err = common.ValidateTimeout(val); err == nil {
					result = val
				} else {
					log.Infof("%s. default value %d used", err, defaultValue)
				}
			} else {
				log.Infof("Cannot convert %s to int for param %s, default value %d used", v[0], name, defaultValue)
			}
			break
		}
	}
	return result
}
