package handlers

import (
	"fmt"
	"net/http"

	"get.porter.sh/porter/pkg/porter"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/generator"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/helpers"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/models"
)

// NewArcHandler is the router for Arc Template generation requests
func NewArcHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(models.BundleCtx)
	r.Get("/*", arcHandler)
	return r
}

func arcHandler(w http.ResponseWriter, r *http.Request) {
	bundleContext := r.Context().Value(models.BundleContext).(*models.Bundle)

	opts := porter.BundlePullOptions{
		InsecureRegistry: bundleContext.InsecureRegistry,
		Force:            bundleContext.Force,
		Tag:              bundleContext.Ref,
	}

	options := common.BundleDetails{
		BundleLoc: "",
		Options: common.Options{
			Indent:            true,
			OutputWriter:      w,
			Simplify:          bundleContext.Simplyfy,
			ReplaceKubeconfig: bundleContext.ReplaceKubeconfig,
			BundlePullOptions: &opts,
			Timeout:           bundleContext.Timeout,
			Debug:             bundleContext.Debug,
		},
	}
	generatedTemplate, _, err := generator.GenerateArcTemplate(options)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to generate Arc template for image: %s error: %v", bundleContext.Ref, err)))
	}
	err = common.WriteOutput(w, generatedTemplate, options.Indent)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to write Arc template to response for image: %s error: %v", bundleContext.Ref, err)))
	}

}
