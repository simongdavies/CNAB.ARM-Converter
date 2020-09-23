package handlers

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/models"
)

// NewCustomResourceHandler is the router for Custom Resource requests
func NewCustomResourceHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(models.BundleCtx)
	r.Get("/*", getCustomResourceHandler)
	r.Put("/*", putCustomResourceHandler)
	r.Post("/*", postCustomResourceHandler)
	r.Delete("/*", deleteCustomResourceHandler)
	return r
}

func getCustomResourceHandler(w http.ResponseWriter, r *http.Request) {
}

func putCustomResourceHandler(w http.ResponseWriter, r *http.Request) {
}

func postCustomResourceHandler(w http.ResponseWriter, r *http.Request) {
}

func deleteCustomResourceHandler(w http.ResponseWriter, r *http.Request) {
}
