package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/handlers"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/models"
	log "github.com/sirupsen/logrus"
)

// Listen starts a new HTTP Listener

func Listen() {

	port, exists := os.LookupEnv("LISTENER_PORT")
	if !exists {
		port = "8080"
	}
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(common.SetOriginalRequestURI)
	router.Use(middleware.Logger)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(middleware.Recoverer)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	router.Handle(models.TemplateGeneratorPath+"/*", handlers.NewTemplateHandler())
	router.Handle(models.NestedResourceGeneratorPath+"/*", handlers.NewNestedDeploymentHandler())
	router.Handle(models.UIDefPath+"/*", handlers.NewUIHandler())
	router.Handle(models.RedirectPath+"/*", handlers.NewRedirectHandler())
	router.Handle(models.UIRedirectPath+"/*", handlers.NewUIRedirectHandler())
	log.Infof("Starting to listen on port  %s", port)
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), router)
	if err != nil {
		log.Fatalf("Error running HTTP Server %v", err)
	}
}
