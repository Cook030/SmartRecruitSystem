package main

import (
	"fmt"
	"log"
	"net"

	"logic-grpc-service/config"
	"logic-grpc-service/pb"
	rds "logic-grpc-service/redis"
	"logic-grpc-service/repository"
	"logic-grpc-service/service"

	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect mysql: %v", err)
	}
	log.Println("MySQL connected")

	redisClient := rds.NewClient(cfg.Redis)
	rds.GlobalClient = redisClient

	repo := repository.NewRepository(db, redisClient)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterAuthServiceServer(grpcServer, service.NewAuthService(repo))
	pb.RegisterJobServiceServer(grpcServer, service.NewJobService(repo))
	pb.RegisterCandidateServiceServer(grpcServer, service.NewCandidateService(repo, cfg))
	pb.RegisterChatServiceServer(grpcServer, service.NewChatService(repo, cfg))
	pb.RegisterDashboardServiceServer(grpcServer, service.NewDashboardService(repo))
	pb.RegisterTaskServiceServer(grpcServer, service.NewTaskService(repo, cfg))

	log.Printf("Logic gRPC service listening on :%d", cfg.GRPC.Port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
