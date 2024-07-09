package main

import (
	"context"
	"eduseal/internal/apigw/apiv1"
	"eduseal/internal/apigw/db"
	"eduseal/internal/apigw/httpserver"
	"eduseal/pkg/configuration"
	"eduseal/pkg/grpcclient"
	"eduseal/pkg/kvclient"
	"eduseal/pkg/logger"
	"eduseal/pkg/trace"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type service interface {
	Close(ctx context.Context) error
}

func main() {
	var wg sync.WaitGroup
	ctx := context.Background()

	services := make(map[string]service)

	cfg, err := configuration.Parse(ctx, logger.NewSimple("Configuration"))
	if err != nil {
		panic(err)
	}

	log, err := logger.New("eduseal_apigw", cfg.Common.Log.FolderPath, cfg.Common.Production)
	if err != nil {
		panic(err)
	}
	tracer, err := trace.New(ctx, cfg, log, "eduseal", "apigw")
	if err != nil {
		panic(err)
	}

	grpcClient, err := grpcclient.New(ctx, cfg, log.New("grpcclient"))
	if err != nil {
		panic(err)
	}

	kvClient, err := kvclient.New(ctx, cfg, tracer, log.New("kvclient"))
	services["kvClient"] = kvClient
	if err != nil {
		panic(err)
	}

	dbService, err := db.New(ctx, cfg, tracer, log.New("db"))
	services["dbService"] = dbService
	if err != nil {
		panic(err)
	}

	apiv1Client, err := apiv1.New(ctx, kvClient, grpcClient, dbService, tracer, cfg, log.New("apiv1"))
	if err != nil {
		panic(err)
	}
	httpService, err := httpserver.New(ctx, cfg, apiv1Client, tracer, log.New("httpserver"))
	services["httpService"] = httpService
	if err != nil {
		panic(err)
	}

	// Handle sigterm and await termChan signal
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	<-termChan // Blocks here until interrupted

	mainLog := log.New("main")
	mainLog.Info("HALTING SIGNAL!")

	for serviceName, service := range services {
		if err := service.Close(ctx); err != nil {
			mainLog.Trace("serviceName", serviceName, "error", err)
		}
	}

	if err := tracer.Shutdown(ctx); err != nil {
		mainLog.Error(err, "Tracer shutdown")
	}

	wg.Wait() // Block here until are workers are done

	mainLog.Info("Stopped")
}
