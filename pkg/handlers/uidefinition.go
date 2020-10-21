package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"get.porter.sh/porter/pkg/porter"
	bundledef "github.com/cnabio/cnab-go/bundle"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/generator"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/helpers"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/models"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/template"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/uidefinition"
	log "github.com/sirupsen/logrus"
)

// NewUIHandler is the router for UI definition generation requests
func NewUIHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(models.BundleCtx)
	r.Get("/*", uiHandler)
	return r
}

// NewNestedDeploymentHandler is the router for Nested Resource generation requests
func NewUIRedirectHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(models.BundleCtx)
	r.Get("/*", uiRedirectHandler)
	return r
}

func uiHandler(w http.ResponseWriter, r *http.Request) {
	bundle := r.Context().Value(models.BundleContext).(*models.Bundle)

	opts := porter.BundlePullOptions{
		InsecureRegistry: bundle.InsecureRegistry,
		Force:            bundle.Force,
		Tag:              bundle.Ref,
	}

	options := common.BundleDetails{
		BundleLoc: "",
		Options: common.Options{
			Indent:                true,
			OutputWriter:          w,
			Simplify:              bundle.Simplyfy,
			ReplaceKubeconfig:     bundle.ReplaceKubeconfig,
			GenerateUI:            true,
			UIWriter:              w,
			BundlePullOptions:     &opts,
			Timeout:               bundle.Timeout,
			IncludeCustomResource: bundle.IncludeCustomResource,
			CustomRPTemplate:      bundle.CustomRPTemplate,
		},
	}

	var generatedTemplate *template.Template
	var bundledef *bundledef.Bundle
	var err error

	if options.CustomRPTemplate {
		generatedTemplate, bundledef, err = generator.GenerateCustomRP(options)
	} else {
		generatedTemplate, bundledef, err = generator.GenerateTemplate(options)
	}

	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to generate template for image: %s error: %v", bundle.Ref, err)))
	}
	ui, err := uidefinition.NewCreateUIDefinition(bundledef.Name, bundledef.Description, generatedTemplate, options.Simplify, options.ReplaceKubeconfig, bundledef.Custom, bundle.CustomRPTemplate, bundle.IncludeCustomResource)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to generate UI Def for image: %s error: %v", bundle.Ref, err)))
	}

	if err := common.WriteOutput(options.UIWriter, ui, options.Indent); err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to Write UI output for image: %s error: %v", bundle.Ref, err)))
	}
}

func uiRedirectHandler(w http.ResponseWriter, r *http.Request) {
	originalRequestUri := r.Context().Value(common.RequestURIContext).(string)
	templateUri := strings.Replace(originalRequestUri, models.UIRedirectPath, models.TemplateGeneratorPath, 1)
	uiURI := strings.Replace(originalRequestUri, models.UIRedirectPath, models.UIDefPath, 1)
	redirectURI := fmt.Sprintf("https://portal.azure.com/#create/Microsoft.Template/uri/%s/createUIDefinitionUri/%s", url.PathEscape(templateUri), url.PathEscape(uiURI))
	log.Infof("Redirecting %s to %s", originalRequestUri, redirectURI)
	http.Redirect(w, r, redirectURI, http.StatusTemporaryRedirect)
}
