// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"fmt"
	"github.com/ATenderholt/dockerlib"
	"github.com/ATenderholt/rainbow-functions/internal/dev"
	"github.com/ATenderholt/rainbow-functions/internal/docker"
	"github.com/ATenderholt/rainbow-functions/internal/domain"
	"github.com/ATenderholt/rainbow-functions/internal/http"
	"github.com/ATenderholt/rainbow-functions/internal/repo"
	"github.com/ATenderholt/rainbow-functions/internal/sqs"
	"github.com/ATenderholt/rainbow-functions/pkg/database"
	"github.com/ATenderholt/rainbow-functions/settings"
	"github.com/go-chi/chi/v5"
	"github.com/google/wire"
	http2 "net/http"
)

import (
	_ "github.com/mattn/go-sqlite3"
)

// Injectors from inject.go:

func InjectApp(cfg *settings.Config) (App, error) {
	database := RealDatabase(cfg)
	layerRepository := repo.NewLayerRepository(database)
	runtimeRepository := repo.NewRuntimeRepository(database)
	layerHandler := http.NewLayerHandler(cfg, layerRepository, runtimeRepository)
	functionRepository := repo.NewFunctionRepository(database)
	manager, err := docker.NewManager(cfg)
	if err != nil {
		return App{}, err
	}
	functionHandler := http.NewFunctionHandler(cfg, functionRepository, layerRepository, runtimeRepository, manager)
	eventSourceRepository := repo.NewEventSourceRepository(database)
	eventSourceHandler := http.NewEventSourceHandler(cfg, eventSourceRepository, functionRepository)
	mux := http.NewChiMux(layerHandler, functionHandler, eventSourceHandler, manager)
	sqsManager := sqs.NewManager(cfg, eventSourceRepository)
	dockerController, err := dockerlib.NewDockerController()
	if err != nil {
		return App{}, err
	}
	service := dev.NewService(dockerController)
	app := NewApp(cfg, mux, manager, sqsManager, functionRepository, service)
	return app, nil
}

// inject.go:

func NewApp(cfg *settings.Config, mux *chi.Mux, docker2 *docker.Manager, sqs2 *sqs.Manager,
	functionRepo domain.FunctionRepository, devService *dev.Service) App {

	srv := &http2.Server{
		Addr:    fmt.Sprintf(":%d", cfg.BasePort),
		Handler: mux,
	}

	return App{
		cfg:          cfg,
		srv:          srv,
		docker:       docker2,
		sqs:          sqs2,
		functionRepo: functionRepo,
		devService:   devService,
	}
}

func RealDatabase(cfg *settings.Config) database.Database {
	return database.RealDatabase{
		Wrapped: cfg.CreateDatabase(),
	}
}

var db = wire.NewSet(
	RealDatabase, repo.NewFunctionRepository, repo.NewLayerRepository, repo.NewRuntimeRepository, repo.NewEventSourceRepository, wire.Bind(new(domain.FunctionRepository), new(*repo.FunctionRepository)), wire.Bind(new(domain.LayerRepository), new(*repo.LayerRepository)), wire.Bind(new(domain.RuntimeRepository), new(*repo.RuntimeRepository)), wire.Bind(new(domain.EventSourceRepository), new(*repo.EventSourceRepository)),
)

var api = wire.NewSet(http.NewFunctionHandler, http.NewLayerHandler, http.NewEventSourceHandler, http.NewChiMux)
