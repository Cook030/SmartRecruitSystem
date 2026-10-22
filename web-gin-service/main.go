package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"web-gin-service/config"
	"web-gin-service/pb"
	"web-gin-service/router"
	"web-gin-service/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("service stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	metricsAddr := os.Getenv("WEB_METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = "127.0.0.1:9464"
	}
	shutdown, err := telemetry.Init(ctx, "recruitment-web", metricsAddr)
	if err != nil {
		return fmt.Errorf("initialize telemetry: %w", err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdown(cleanup); err != nil {
			slog.Warn("telemetry shutdown failed", "error", err)
		}
	}()

	conn, err := grpc.NewClient(cfg.GRPC.LogicAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		return fmt.Errorf("connect gRPC: %w", err)
	}
	defer conn.Close()
	slog.Info("Connected to Logic gRPC service", "address", cfg.GRPC.LogicAddr)

	authClient := pb.NewAuthServiceClient(conn)
	jobClient := pb.NewJobServiceClient(conn)
	candidateClient := pb.NewCandidateServiceClient(conn)
	chatClient := pb.NewChatServiceClient(conn)
	dashboardClient := pb.NewDashboardServiceClient(conn)
	taskClient := pb.NewTaskServiceClient(conn)

	r := router.SetupRouter(authClient, jobClient, candidateClient, chatClient, dashboardClient, taskClient)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("Web Gin service listening", "address", addr)
	server := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-ctx.Done()
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(cleanup); err != nil {
			_ = server.Close()
		}
	}()
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	stop()
	<-stopped
	return nil
}
