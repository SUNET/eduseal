package main

import (
	"context"
	"eduseal/internal/apigw/apiv1"
	"eduseal/internal/apigw/httpserver"
	"eduseal/internal/apigw/stream"
	"eduseal/pkg/configuration"
	"eduseal/pkg/grpcclient"
	"eduseal/pkg/kvclient"
	"eduseal/pkg/logger"
	"eduseal/pkg/metric"
	"eduseal/pkg/trace"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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

	tracer, err := trace.New(ctx, cfg, "eduseal_apigw", log.New("tracer"))
	if err != nil {
		panic(err)
	}

	metric, err := metric.New(ctx, cfg, "eduseal_apigw", log.New("metric"))
	if err != nil {
		panic(err)
	}

	grpcClient, err := grpcclient.New(ctx, cfg, tracer, log.New("grpcclient"))
	if err != nil {
		panic(err)
	}

	kvClient, err := kvclient.New(ctx, cfg, tracer, log.New("kvclient"))
	services["kvClient"] = kvClient
	if err != nil {
		panic(err)
	}

	streamService, err := stream.New(ctx, kvClient, tracer, cfg, log.New("stream"))
	services["streamService"] = streamService
	if err != nil {
		panic(err)
	}

	apiv1Client, err := apiv1.New(ctx, kvClient, grpcClient, streamService, tracer, metric, cfg, log.New("apiv1"))
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

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 1. Stop HTTP server first - this stops accepting new requests and waits for in-flight requests
	if httpService, ok := services["httpService"]; ok {
		mainLog.Info("Shutting down HTTP server - waiting for in-flight requests...")
		if err := httpService.Close(shutdownCtx); err != nil {
			mainLog.Error(err, "HTTP server shutdown")
		}
		mainLog.Info("HTTP server stopped")
		delete(services, "httpService")
	}

	// 2. Close stream service after HTTP server (no new requests can arrive)
	if streamService, ok := services["streamService"]; ok {
		mainLog.Info("Closing stream service...")
		if err := streamService.Close(shutdownCtx); err != nil {
			mainLog.Error(err, "Stream service close")
		}
		mainLog.Info("Stream service closed")
		delete(services, "streamService")
	}

	// 3. Close remaining services
	for serviceName, service := range services {
		mainLog.Info("Closing service", "service", serviceName)
		if err := service.Close(shutdownCtx); err != nil {
			mainLog.Error(err, "Service close", "service", serviceName)
		}
	}

	if err := tracer.Shutdown(shutdownCtx); err != nil {
		mainLog.Error(err, "Tracer shutdown")
	}

	wg.Wait() // Block here until are workers are done

	mainLog.Info("Stopped")
}
