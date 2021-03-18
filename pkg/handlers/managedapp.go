package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"get.porter.sh/porter/pkg/porter"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/generator"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/helpers"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/models"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/uidefinition"
)

const mainTemplateFileName = "mainTemplate.json"
const createUIDefFileName = "createUiDefinition.json"
const viewDefFileName = "viewDefinition.json"

// NewManagedAppHandler is the router for Managed App generation requests
func NewManagedAppHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(models.BundleCtx)
	r.Get("/*", managedAppHandler)
	return r
}

// NewManagedAppDefinitionHandler is the router for Managed App generation requests
func NewManagedAppDefinitionHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(models.BundleCtx)
	r.Get("/*", managedAppDefinitionHandler)
	return r
}

func managedAppHandler(w http.ResponseWriter, r *http.Request) {
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
			Simplify:              bundle.Simplyfy,
			ReplaceKubeconfig:     bundle.ReplaceKubeconfig,
			BundlePullOptions:     &opts,
			Timeout:               bundle.Timeout,
			IncludeCustomResource: true,
			CustomRPTemplate:      true,
			GenerateUI:            true,
		},
	}

	generatedTemplate, bundledef, err := generator.GenerateCustomRP(options)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Error generating customRP template: %w", err)))
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

	ui, err := uidefinition.NewCreateUIDefinition(bundledef.Name, bundledef.Description, generatedTemplate, options.Simplify, options.ReplaceKubeconfig, bundledef.Custom, options.CustomRPTemplate, options.IncludeCustomResource, options.ArcTemplate)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Failed to generate UI definition, %w", err)))
		return
	}

	if err = common.WriteOutput(uiDefFile, ui, options.Indent); err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Failed to write ui definition output, %w", err)))
		return
	}

	viewDefFile, err := zipWriter.Create(viewDefFileName)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Error creating View Def file: %w", err)))
		return
	}

	viewDef := uidefinition.NewViewDefinition(bundledef.Name, bundledef.Description)

	if err = common.WriteOutput(viewDefFile, viewDef, options.Indent); err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Failed to write view definition output, %w", err)))
		return
	}

	if err = zipWriter.Close(); err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Failed to close zip archive, %w", err)))
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment;filename=application.zip")
	_, err = w.Write(buf.Bytes())
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInternalServerErrorFromError(fmt.Errorf("Failed to write managed app zip: %w", err)))
		return
	}

}

func managedAppDefinitionHandler(w http.ResponseWriter, r *http.Request) {
	bundle := r.Context().Value(models.BundleContext).(*models.Bundle)
	originalRequestUri := r.Context().Value(common.RequestURIContext).(string)
	packageUri := strings.Replace(originalRequestUri, models.ManagedAppDefinitionPath, models.ManagedAppPath, 1)
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
	generatedTemplate, _, err := generator.GenerateManagedAppDefinitionTemplate(options, packageUri)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to generate managed app definition template for image: %s error: %v", bundle.Ref, err)))
	}
	err = common.WriteOutput(w, generatedTemplate, options.Indent)
	if err != nil {
		_ = render.Render(w, r, helpers.ErrorInvalidRequestFromError(fmt.Errorf("Failed to write  managed app definition template to response for image: %s error: %v", bundle.Ref, err)))
	}

}
