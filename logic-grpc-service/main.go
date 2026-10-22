package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"logic-grpc-service/config"
	"logic-grpc-service/pb"
	rds "logic-grpc-service/redis"
	"logic-grpc-service/repository"
	"logic-grpc-service/service"
	"logic-grpc-service/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	metricsAddr := os.Getenv("LOGIC_METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = "127.0.0.1:9465"
	}
	shutdown, err := telemetry.Init(ctx, "recruitment-logic", metricsAddr)
	if err != nil {
		return fmt.Errorf("initialize telemetry: %w", err)
	}
	telemetry.InitBusinessCounters(ctx, cfg.AI.Model)
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdown(cleanup); err != nil {
			log.Printf("telemetry shutdown: %v", err)
		}
	}()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return fmt.Errorf("connect mysql: %w", err)
	}
	log.Println("MySQL connected")
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	if err := db.Use(telemetry.DatabasePlugin{}); err != nil {
		return err
	}
	pool, err := telemetry.RegisterDBPool(sqlDB)
	if err != nil {
		return err
	}
	defer pool.Unregister()

	redisClient := rds.NewClient(cfg.Redis)
	rds.GlobalClient = redisClient

	repo := repository.NewRepository(db, redisClient)
	stopMetrics, err := repo.StartTaskMetrics(ctx)
	if err != nil {
		return err
	}
	defer stopMetrics()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()), grpc.UnaryInterceptor(telemetry.UnaryServerInterceptor))

	pb.RegisterAuthServiceServer(grpcServer, service.NewAuthService(repo))
	pb.RegisterJobServiceServer(grpcServer, service.NewJobService(repo))
	pb.RegisterCandidateServiceServer(grpcServer, service.NewCandidateService(repo, cfg))
	pb.RegisterChatServiceServer(grpcServer, service.NewChatService(repo, cfg))
	pb.RegisterDashboardServiceServer(grpcServer, service.NewDashboardService(repo))
	tasks := service.NewTaskService(ctx, repo, cfg)
	defer tasks.Close()
	pb.RegisterTaskServiceServer(grpcServer, tasks)

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-ctx.Done()
		done := make(chan struct{})
		go func() { grpcServer.GracefulStop(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			grpcServer.Stop()
		}
	}()
	log.Printf("Logic gRPC service listening on :%d", cfg.GRPC.Port)
	if err := grpcServer.Serve(lis); err != nil {
		if !errors.Is(err, grpc.ErrServerStopped) {
			return err
		}
	}
	stop()
	<-stopped
	return nil
}
