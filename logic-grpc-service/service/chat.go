package service

import (
	"context"
	"fmt"
	"strings"

	"logic-grpc-service/ai"
	"logic-grpc-service/config"
	"logic-grpc-service/model"
	"logic-grpc-service/pb"
	"logic-grpc-service/repository"

	"github.com/cloudwego/eino/schema"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChatService struct {
	pb.UnimplementedChatServiceServer
	repo *repository.Repository
	cfg  *config.Config
}

func NewChatService(repo *repository.Repository, cfg *config.Config) *ChatService {
	return &ChatService{repo: repo, cfg: cfg}
}

func (s *ChatService) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "message content is required")
	}

	bizContext := s.buildBusinessContext()

	enrichedMessage := fmt.Sprintf("以下是系统当前的真实业务数据：\n%s\n\nHR 的问题：%s", bizContext, req.Content)

	historyMsgs, _ := s.repo.GetChatHistory(req.HrId)
	var history []*schema.Message
	for _, msg := range historyMsgs {
		role := schema.User
		if msg.Role == "assistant" {
			role = schema.Assistant
		}
		history = append(history, &schema.Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	reply, err := ai.Chat(s.cfg, history, enrichedMessage)
	if err != nil {
		return nil, status.Error(codes.Internal, "AI chat failed: "+err.Error())
	}

	userMsg := &model.ChatMessage{HrID: req.HrId, Role: "user", Content: req.Content}
	_ = s.repo.CreateChatMessage(userMsg)

	aiMsg := &model.ChatMessage{HrID: req.HrId, Role: "assistant", Content: reply}
	_ = s.repo.CreateChatMessage(aiMsg)

	return &pb.SendMessageResponse{Reply: reply}, nil
}

func (s *ChatService) GetChatHistory(ctx context.Context, req *pb.GetChatHistoryRequest) (*pb.GetChatHistoryResponse, error) {
	msgs, err := s.repo.GetChatHistory(req.HrId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get chat history")
	}

	var list []*pb.ChatMessageInfo
	for _, msg := range msgs {
		list = append(list, &pb.ChatMessageInfo{
			Id:        msg.ID,
			HrId:      msg.HrID,
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &pb.GetChatHistoryResponse{Messages: list}, nil
}

func (s *ChatService) buildBusinessContext() string {
	var parts []string

	// 投递总数
	totalApps, err := s.repo.CountAllApplications()
	if err == nil {
		parts = append(parts, fmt.Sprintf("- 系统总投递数: %d", totalApps))
	}

	// 候选人总数
	totalCandidates, err := s.repo.CountCandidateProfiles()
	if err == nil {
		parts = append(parts, fmt.Sprintf("- 已完善资料的候选人总数: %d", totalCandidates))
	}

	// 各岗位投递统计（热度分析）
	jobStats, err := s.repo.GetAllJobsWithApplicationCount()
	if err == nil && len(jobStats) > 0 {
		parts = append(parts, "- 各岗位投递情况（按投递数降序排列，即热度排行）:")
		for _, stat := range jobStats {
			title, _ := stat["title"].(string)
			count, _ := stat["application_count"].(int64)
			location, _ := stat["location"].(string)
			salary, _ := stat["salary"].(string)
			parts = append(parts, fmt.Sprintf("  - %s（地点: %s, 薪资: %s）: %d 人投递", title, location, salary, count))
		}
	}

	// 学历分布
	eduDist, err := s.repo.GetEducationDistribution()
	if err == nil && len(eduDist) > 0 {
		parts = append(parts, "- 候选人学历分布:")
		for _, ed := range eduDist {
			education, _ := ed["education"].(string)
			count, _ := ed["count"].(int64)
			parts = append(parts, fmt.Sprintf("  - %s: %d 人", education, count))
		}
	}

	// 候选人档案列表（姓名、学历、院校、技能）
	profiles, err := s.repo.GetAllCandidateProfiles()
	if err == nil && len(profiles) > 0 {
		parts = append(parts, "- 候选人档案列表:")
		for _, p := range profiles {
			parts = append(parts, fmt.Sprintf("  - 姓名: %s, 学历: %s, 院校: %s, 技能: %s",
				p.Name, p.Education, p.University, p.Skills))
		}
	}

	// 候选人投递明细（谁投递了什么岗位）
	appDetails, err := s.repo.GetCandidateApplicationDetails()
	if err == nil && len(appDetails) > 0 {
		parts = append(parts, "- 候选人投递明细:")
		for _, d := range appDetails {
			parts = append(parts, fmt.Sprintf("  - %s（%s, %s, 技能: %s）→ 投递岗位: %s",
				d.CandidateName, d.Education, d.University, d.Skills, d.JobTitle))
		}
	}

	return strings.Join(parts, "\n")
}
