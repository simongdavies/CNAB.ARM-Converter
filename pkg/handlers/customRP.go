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

// NewCustomRPHandler is the router for Custom Resource requests
func NewCustomRPHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(models.BundleCtx)
	r.Get("/*", getCustomRPHandler)
	return r
}

func getCustomRPHandler(w http.ResponseWriter, r *http.Request) {

	bundle := r.Context().Value(models.BundleContext).(*models.Bundle)

	opts := porter.BundlePullOptions{
		InsecureRegistry: bundle.InsecureRegistry,
		Force:            bundle.Force,
		Tag:              bundle.Ref,
	}

	options := common.BundleDetails{
		BundleLoc: "",
		Options: common.Options{
			Indent:            true,
			OutputWriter:      w,
			Simplify:          bundle.Simplyfy,
			ReplaceKubeconfig: bundle.ReplaceKubeconfig,
			BundlePullOptions: &opts,
			Timeout:           bundle.Timeout,
		},
	}
	generatedCustomRPTemplate, _, err := generator.GenerateCustomRP(options)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to generate custom RP template for image: %s error: %v", bundle.Ref, err)))
	}
	err = common.WriteOutput(w, generatedCustomRPTemplate, options.Indent)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to write custom RP template to response for image: %s error: %v", bundle.Ref, err)))
	}
}
