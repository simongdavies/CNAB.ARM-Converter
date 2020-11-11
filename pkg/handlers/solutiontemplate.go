package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"

	"get.porter.sh/porter/pkg/porter"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/generator"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/helpers"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/models"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/uidefinition"
)

// NewSolutionTemplateHandler is the router for Solution Template generation requests
func NewSolutionTemplateHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(models.BundleCtx)
	r.Get("/*", solutionTemplateHandler)
	return r
}

func solutionTemplateHandler(w http.ResponseWriter, r *http.Request) {
	bundle := r.Context().Value(models.BundleContext).(*models.Bundle)

	opts := porter.BundlePullOptions{
		InsecureRegistry: bundle.InsecureRegistry,
		Force:            bundle.Force,
		Tag:              bundle.Ref,
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	templateFile, err := zipWriter.Create(mainTemplateFileName)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Error creating template file: %w", err)))
		return
	}

	options := common.BundleDetails{
		BundleLoc: "",
		Options: common.Options{
			Indent:                true,
			Simplify:              true,
			ReplaceKubeconfig:     true,
			BundlePullOptions:     &opts,
			Timeout:               bundle.Timeout,
			IncludeCustomResource: false,
			CustomRPTemplate:      false,
			GenerateUI:            true,
		},
	}

	generatedTemplate, bundledef, err := generator.GenerateTemplate(options)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Error generating solution template ARM template: %w", err)))
		return
	}

	err = common.WriteOutput(templateFile, generatedTemplate, options.Indent)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Error writing output file: %w", err)))
		return
	}

	uiDefFile, err := zipWriter.Create(createUIDefFileName)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Error creating UI Def file: %w", err)))
		return
	}

	ui, err := uidefinition.NewCreateUIDefinition(bundledef.Name, bundledef.Description, generatedTemplate, options.Simplify, options.ReplaceKubeconfig, bundledef.Custom, options.CustomRPTemplate, options.IncludeCustomResource)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Failed to generate UI definition, %w", err)))
		return
	}

	if err = common.WriteOutput(uiDefFile, ui, options.Indent); err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Failed to write ui definition output, %w", err)))
		return
	}

	if err = zipWriter.Close(); err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Failed to close zip archive, %w", err)))
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment;filename=solutionTemplate.zip")
	_, err = w.Write(buf.Bytes())
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Failed to write solution template zip: %w", err)))
		return
	}

}
