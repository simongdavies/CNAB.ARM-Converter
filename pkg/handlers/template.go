package handlers

import (
	"fmt"
	"net/http"

	"get.porter.sh/porter/pkg/porter"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/generator"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/helpers"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/models"
)

// NewTemplateHandler is the router for Template requests
func NewTemplateHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(models.BundleCtx)
	r.Get("/*", templateHandler)
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
		BundleLoc:         "",
		Indent:            true,
		Writer:            w,
		Simplify:          bundle.Simplyfy,
		BundlePullOptions: &opts,
	}
	err := generator.GenerateTemplate(options)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to generate template for image: %s error: %v", bundle.Ref, err)))
	}
}
