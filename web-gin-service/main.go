package main

import (
	"fmt"
	"log"

	"web-gin-service/config"
	"web-gin-service/pb"
	"web-gin-service/router"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	conn, err := grpc.NewClient(cfg.GRPC.LogicAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect gRPC: %v", err)
	}
	defer conn.Close()
	log.Printf("Connected to Logic gRPC service at %s", cfg.GRPC.LogicAddr)

	authClient := pb.NewAuthServiceClient(conn)
	jobClient := pb.NewJobServiceClient(conn)
	candidateClient := pb.NewCandidateServiceClient(conn)
	chatClient := pb.NewChatServiceClient(conn)
	dashboardClient := pb.NewDashboardServiceClient(conn)
	taskClient := pb.NewTaskServiceClient(conn)

	r := router.SetupRouter(authClient, jobClient, candidateClient, chatClient, dashboardClient, taskClient)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Web Gin service listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
