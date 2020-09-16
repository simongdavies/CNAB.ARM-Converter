package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"get.porter.sh/porter/pkg/porter"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/generator"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/helpers"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/models"
	log "github.com/sirupsen/logrus"
)

// NewTemplateHandler is the router for Template generation requests
func NewTemplateHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(models.BundleCtx)
	r.Get("/*", templateHandler)
	return r
}

// NewNestedDeploymentHandler is the router for Nested Resource generation requests
func NewNestedDeploymentHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(models.BundleCtx)
	r.Get("/*", nestedDeploymentHandler)
	return r
}

// NewNestedDeploymentHandler is the router for Nested Resource generation requests
func NewRedirectHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(models.BundleCtx)
	r.Get("/*", redirectHandler)
	return r
}

func templateHandler(w http.ResponseWriter, r *http.Request) {
	bundle := r.Context().Value(models.BundleContext).(*models.Bundle)

	opts := porter.BundlePullOptions{
		InsecureRegistry: bundle.InsecureRegistry,
		Force:            bundle.Force,
		Tag:              bundle.Ref,
	}

	options := generator.GenerateTemplateOptions{
		BundleLoc: "",
		GenerateOptions: generator.GenerateOptions{
			Indent:            true,
			Writer:            w,
			Simplify:          bundle.Simplyfy,
			ReplaceKubeconfig: bundle.ReplaceKubeconfig,
			BundlePullOptions: &opts,
			Timeout:           bundle.Timeout,
		},
	}
	err := generator.GenerateTemplate(options)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to generate template for image: %s error: %v", bundle.Ref, err)))
	}
}

func nestedDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	bundle := r.Context().Value(models.BundleContext).(*models.Bundle)
	originalRequestUri := r.Context().Value(common.RequestURIContext).(string)

	opts := porter.BundlePullOptions{
		InsecureRegistry: bundle.InsecureRegistry,
		Force:            bundle.Force,
		Tag:              bundle.Ref,
	}

	options := generator.GenerateNestedDeploymentOptions{
		GenerateOptions: generator.GenerateOptions{
			Indent:            true,
			Writer:            w,
			Simplify:          bundle.Simplyfy,
			ReplaceKubeconfig: bundle.ReplaceKubeconfig,
			BundlePullOptions: &opts,
		},
	}

	options.Uri = strings.Replace(originalRequestUri, models.NestedResourceGeneratorPath, models.TemplateGeneratorPath, 1)

	err := generator.GenerateNestedDeployment(options)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to generate nested deployment for image: %s error: %v", bundle.Ref, err)))
	}
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	originalRequestUri := r.Context().Value(common.RequestURIContext).(string)
	templateUri := strings.Replace(originalRequestUri, models.RedirectPath, models.TemplateGeneratorPath, 1)
	redirectURI := fmt.Sprintf("%s/%s", "https://portal.azure.com/#create/Microsoft.Template/uri", url.PathEscape(templateUri))
	log.Infof("Redirecting %s to %s", originalRequestUri, redirectURI)
	http.Redirect(w, r, redirectURI, http.StatusTemporaryRedirect)
}
