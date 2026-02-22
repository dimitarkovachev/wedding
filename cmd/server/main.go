package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	"github.com/dimitarkovachev/wedding/internal/admin"
	"github.com/dimitarkovachev/wedding/internal/api"
	"github.com/dimitarkovachev/wedding/internal/config"
	"github.com/dimitarkovachev/wedding/internal/middleware"
	"github.com/dimitarkovachev/wedding/internal/seed"
	"github.com/dimitarkovachev/wedding/internal/store"
)

func main() {
	cfg := config.Load()

	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)

	gin.SetMode(cfg.GinMode)

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0755); err != nil {
		log.WithError(err).Fatal("failed to create db directory")
	}

	bboltStore, err := store.NewBBoltStore(cfg.DBPath)
	if err != nil {
		log.WithError(err).Fatal("failed to open bbolt store")
	}
	defer bboltStore.Close()

	if err := seed.LoadFromFile(cfg.SeedFile, bboltStore); err != nil {
		log.WithError(err).Fatal("failed to seed data")
	}

	swagger, err := api.GetSwagger()
	if err != nil {
		log.WithError(err).Fatal("failed to load embedded swagger spec")
	}

	validator, err := middleware.NewOpenAPIValidator(swagger)
	if err != nil {
		log.WithError(err).Fatal("failed to create openapi validator")
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.NewRateLimiter(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst))
	r.Use(validator)

	handler := api.NewHandler(bboltStore)
	api.RegisterHandlers(r, handler)

	srv := &http.Server{
		Handler: r,
		Addr:    net.JoinHostPort("0.0.0.0", cfg.Port),
	}

	adminRouter := gin.New()
	adminRouter.Use(gin.Recovery())

	adminHandler := admin.NewHandler(bboltStore)
	admin.RegisterHandlers(adminRouter, adminHandler)

	adminSrv := &http.Server{
		Handler: adminRouter,
		Addr:    net.JoinHostPort("0.0.0.0", cfg.AdminPort),
	}

	go func() {
		log.WithField("addr", srv.Addr).Info("starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("server error")
		}
	}()

	go func() {
		log.WithField("addr", adminSrv.Addr).Info("starting admin server")
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("admin server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.WithField("signal", fmt.Sprintf("%v", sig)).Info("shutting down servers")

	if err := srv.Close(); err != nil {
		log.WithError(err).Error("server close error")
	}
	if err := adminSrv.Close(); err != nil {
		log.WithError(err).Error("admin server close error")
	}
}
