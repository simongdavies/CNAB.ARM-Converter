package handlers

import (
	"fmt"
	"net/http"

	"get.porter.sh/porter/pkg/porter"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/helpers"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/models"
)

// NewBundleHandler is the router for Template generation requests
func NewBundleHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(models.BundleCtx)
	r.Get("/*", bundleHandler)
	return r
}

func bundleHandler(w http.ResponseWriter, r *http.Request) {
	bundleContext := r.Context().Value(models.BundleContext).(*models.Bundle)

	opts := porter.BundlePullOptions{
		InsecureRegistry: bundleContext.InsecureRegistry,
		Force:            bundleContext.Force,
		Tag:              bundleContext.Ref,
	}

	bundle, _, err := common.PullBundle(&opts)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to get bundle.json for image: %s error: %v", bundleContext.Ref, err)))
	}

	err = common.WriteOutput(w, bundle, true)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to write bundle.json to response for image: %s error: %v", bundleContext.Ref, err)))
	}

}
