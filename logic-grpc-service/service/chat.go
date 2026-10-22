package service

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"logic-grpc-service/ai"
	"logic-grpc-service/config"
	"logic-grpc-service/model"
	"logic-grpc-service/pb"
	"logic-grpc-service/repository"

	"github.com/go-sql-driver/mysql"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type ChatService struct {
	pb.UnimplementedChatServiceServer
	repo *repository.Repository
	cfg  *config.Config
}

var chatRequestID = regexp.MustCompile(`^[A-Za-z0-9_-]{16,64}$`)

func chatError(err error) error {
	var duplicate *mysql.MySQLError
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return status.Error(codes.NotFound, "会话不存在或无权访问")
	case errors.Is(err, repository.ErrChatBusy):
		return status.Error(codes.Aborted, "此会话仍在生成回复，请稍后重试")
	case errors.Is(err, repository.ErrChatConflict), errors.As(err, &duplicate) && duplicate.Number == 1062:
		return status.Error(codes.AlreadyExists, "请求 ID 已用于其他问题或会话")
	case errors.Is(err, repository.ErrChatAttemptExpired):
		return status.Error(codes.Aborted, "本次生成已过期，请重试")
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return status.FromContextError(err).Err()
	default:
		slog.Error("chat database operation failed", "error", err)
		return status.Error(codes.Internal, "对话操作失败，请稍后重试")
	}
}

func NewChatService(repo *repository.Repository, cfg *config.Config) *ChatService {
	return &ChatService{repo: repo, cfg: cfg}
}

func (s *ChatService) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (_ *pb.SendMessageResponse, retErr error) {
	if req.HrId == 0 {
		return nil, status.Error(codes.Unauthenticated, "HR identity is required")
	}
	if strings.TrimSpace(req.Content) == "" || utf8.RuneCountInString(req.Content) > 4000 {
		return nil, status.Error(codes.InvalidArgument, "message must contain 1..4000 characters")
	}
	if req.ConversationId == 0 || !chatRequestID.MatchString(req.RequestId) {
		return nil, status.Error(codes.InvalidArgument, "conversation_id and a 16..64 character request_id are required")
	}
	started, replayed := time.Now(), false
	defer func() {
		slog.InfoContext(ctx, "chat request finished", "hr_id", req.HrId, "conversation_id", req.ConversationId,
			"request_id", req.RequestId, "code", status.Code(retErr).String(), "duration_ms", time.Since(started).Milliseconds(), "replayed", replayed)
	}()
	ctx, cancel := context.WithTimeout(ctx, ai.ChatTimeout)
	defer cancel()
	request, err := s.repo.BeginChatRequest(ctx, req.HrId, req.ConversationId, req.RequestId, req.Content)
	if err != nil {
		return nil, chatError(err)
	}
	if request.Status == "completed" {
		replayed = true
		return &pb.SendMessageResponse{Reply: request.Reply}, nil
	}
	completed := false
	defer func() {
		if completed {
			return
		}
		cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer stop()
		if err := s.repo.FailChatRequest(cleanup, request); err != nil {
			slog.ErrorContext(cleanup, "failed to release chat request", "request_id", req.RequestId, "error", err)
		}
	}()
	input, err := s.repo.LoadChatContext(ctx, req.HrId, req.ConversationId)
	if err != nil {
		return nil, chatError(err)
	}
	result, err := ai.Chat(ctx, s.cfg, s.repo, req.HrId, input, req.Content)
	if err != nil {
		if errors.Is(err, ai.ErrTokenBudgetExceeded) {
			return nil, status.Error(codes.ResourceExhausted, "本次请求的 token 额度不足，已停止模型调用")
		}
		if ctx.Err() != nil {
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		slog.ErrorContext(ctx, "AI chat failed", "hr_id", req.HrId, "error", err)
		return nil, status.Error(codes.Internal, "AI chat failed; please retry")
	}
	// Persist a generated answer even if the browser disconnects at this point.
	// Retrying the same request ID can then recover it without another model call.
	commitCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer stop()
	if err := s.repo.CompleteChatRequest(commitCtx, request, result.Reply, result.ContextNotes, result.ContextUpdate); err != nil {
		return nil, chatError(err)
	}
	completed = true

	return &pb.SendMessageResponse{Reply: result.Reply}, nil
}

func (s *ChatService) GetChatHistory(ctx context.Context, req *pb.GetChatHistoryRequest) (*pb.GetChatHistoryResponse, error) {
	if req.HrId == 0 {
		return nil, status.Error(codes.Unauthenticated, "HR identity is required")
	}
	if req.ConversationId == 0 || req.Limit < 0 || req.Limit > 100 {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required; limit must be 1..100")
	}
	limit := int(req.Limit)
	if limit == 0 {
		limit = 40
	}
	msgs, err := s.repo.GetChatHistory(ctx, req.HrId, req.ConversationId, req.BeforeId, limit+1)
	if err != nil {
		return nil, chatError(err)
	}
	var next uint64
	if len(msgs) > limit {
		msgs = msgs[1:]
		next = msgs[0].ID
	}

	var list []*pb.ChatMessageInfo
	for _, msg := range msgs {
		list = append(list, &pb.ChatMessageInfo{
			Id:             msg.ID,
			HrId:           msg.HrID,
			Role:           msg.Role,
			Content:        msg.Content,
			CreatedAt:      msg.CreatedAt.Format("2006-01-02 15:04:05"),
			ConversationId: msg.ConversationID,
		})
	}
	return &pb.GetChatHistoryResponse{Messages: list, NextBeforeId: next}, nil
}

func conversationInfo(c model.ChatConversation) *pb.ChatConversationInfo {
	return &pb.ChatConversationInfo{Id: c.ID, Title: c.Title, CreatedAt: c.CreatedAt.Format(time.RFC3339), UpdatedAt: c.UpdatedAt.Format(time.RFC3339)}
}

func (s *ChatService) CreateConversation(ctx context.Context, req *pb.CreateConversationRequest) (*pb.ChatConversationInfo, error) {
	if req.HrId == 0 {
		return nil, status.Error(codes.Unauthenticated, "HR identity is required")
	}
	title := strings.TrimSpace(req.Title)
	if utf8.RuneCountInString(title) > 80 {
		return nil, status.Error(codes.InvalidArgument, "title must be at most 80 characters")
	}
	if title == "" {
		title = "新对话"
	}
	c, err := s.repo.CreateChatConversation(ctx, req.HrId, title)
	if err != nil {
		return nil, chatError(err)
	}
	return conversationInfo(*c), nil
}

func (s *ChatService) ListConversations(ctx context.Context, req *pb.ListConversationsRequest) (*pb.ListConversationsResponse, error) {
	if req.HrId == 0 {
		return nil, status.Error(codes.Unauthenticated, "HR identity is required")
	}
	items, err := s.repo.ListChatConversations(ctx, req.HrId, req.BeforeId)
	if err != nil {
		return nil, chatError(err)
	}
	response := &pb.ListConversationsResponse{}
	if len(items) > 50 {
		items = items[:50]
		response.NextBeforeId = items[49].ID
	}
	for _, c := range items {
		response.Conversations = append(response.Conversations, conversationInfo(c))
	}
	return response, nil
}

func (s *ChatService) DeleteConversation(ctx context.Context, req *pb.DeleteConversationRequest) (*pb.DeleteConversationResponse, error) {
	if req.HrId == 0 {
		return nil, status.Error(codes.Unauthenticated, "HR identity is required")
	}
	if req.ConversationId == 0 {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}
	if err := s.repo.DeleteChatConversation(ctx, req.HrId, req.ConversationId); err != nil {
		return nil, chatError(err)
	}
	return &pb.DeleteConversationResponse{}, nil
}
