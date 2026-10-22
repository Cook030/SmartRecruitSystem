package router

import (
	"log/slog"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/pb"
	"web-gin-service/telemetry"

	"github.com/gin-gonic/gin"
)

func SetupRouter(authClient pb.AuthServiceClient, jobClient pb.JobServiceClient,
	candidateClient pb.CandidateServiceClient, chatClient pb.ChatServiceClient,
	dashboardClient pb.DashboardServiceClient, taskClient pb.TaskServiceClient) *gin.Engine {

	gin.DefaultWriter = slog.NewLogLogger(slog.Default().Handler(), slog.LevelDebug).Writer()
	gin.DefaultErrorWriter = slog.NewLogLogger(slog.Default().Handler(), slog.LevelError).Writer()
	r := gin.New()
	r.Use(otelgin.Middleware("recruitment-web"), telemetry.HTTPMetrics(), gin.Recovery())
	r.Use(middleware.CORSMiddleware())

	authHandler := handler.NewAuthHandler(authClient)
	jobHandler := handler.NewJobHandler(jobClient, candidateClient)
	candidateHandler := handler.NewCandidateHandler(candidateClient)
	chatHandler := handler.NewChatHandler(chatClient)
	dashboardHandler := handler.NewDashboardHandler(dashboardClient)
	taskHandler := handler.NewTaskHandler(taskClient)

	api := r.Group("/api")

	api.POST("/hr/register", authHandler.HrRegister)
	api.POST("/hr/login", authHandler.HrLogin)
	api.POST("/candidate/register", authHandler.CandidateRegister)
	api.POST("/candidate/login", authHandler.CandidateLogin)
	api.GET("/jobs", jobHandler.ListPublicJobs)

	api.GET("/tasks/:id", middleware.JWTAuth(""), taskHandler.GetTask)

	hr := api.Group("/hr")
	hr.Use(middleware.JWTAuth("hr"))
	{
		hr.GET("/dashboard/overview", dashboardHandler.GetOverview)
		hr.GET("/dashboard/job-trend", dashboardHandler.GetJobTrend)
		hr.GET("/dashboard/application-trend", dashboardHandler.GetApplicationTrend)
		hr.GET("/dashboard/hot-jobs", dashboardHandler.GetHotJobs)
		hr.GET("/dashboard/hot-skills", dashboardHandler.GetHotSkills)
		hr.GET("/dashboard/candidate-funnel", dashboardHandler.GetCandidateFunnel)
		hr.GET("/dashboard/ai-match-stats", dashboardHandler.GetAIMatchStats)
		hr.POST("/dashboard/refresh", taskHandler.RefreshDashboard)
		hr.POST("/jobs", jobHandler.CreateJob)
		hr.PUT("/jobs/:id", jobHandler.UpdateJob)
		hr.PUT("/jobs/:id/offline", jobHandler.OfflineJob)
		hr.GET("/jobs", jobHandler.ListHrJobs)
		hr.GET("/jobs/:id/applications", jobHandler.ListJobApplications)
		hr.GET("/applications/:id/detail", jobHandler.GetApplicationDetail)
		hr.POST("/applications/:id/match", taskHandler.TriggerAIMatch)
		hr.POST("/chat", chatHandler.SendMessage)
		hr.GET("/chat/history", chatHandler.GetChatHistory)
		hr.POST("/chat/conversations", chatHandler.CreateConversation)
		hr.GET("/chat/conversations", chatHandler.ListConversations)
		hr.DELETE("/chat/conversations/:id", chatHandler.DeleteConversation)
	}

	candidate := api.Group("/candidate")
	candidate.Use(middleware.JWTAuth("candidate"))
	{
		candidate.GET("/profile", candidateHandler.GetProfile)
		candidate.PUT("/profile", candidateHandler.UpdateProfile)
		candidate.POST("/resume/upload", candidateHandler.UploadResume)
		candidate.GET("/resume", candidateHandler.GetResume)
		candidate.DELETE("/resume", candidateHandler.DeleteResume)
		candidate.POST("/apply/:jobId", candidateHandler.ApplyJob)
		candidate.GET("/applications", candidateHandler.ListMyApplications)
		candidate.POST("/resume/:id/parse", taskHandler.TriggerResumeParse)
		candidate.GET("/resume/:id/parse-result", taskHandler.GetResumeParseResult)
		candidate.DELETE("/resume/:id/parse-result", taskHandler.DeleteResumeParseResult)
	}

	return r
}
