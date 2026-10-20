package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"
	"web-gin-service/middleware"
	"web-gin-service/pb"
	"web-gin-service/response"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChatHandler struct {
	chatClient pb.ChatServiceClient
}

func NewChatHandler(client pb.ChatServiceClient) *ChatHandler {
	return &ChatHandler{chatClient: client}
}

type sendMessageRequest struct {
	Content        string `json:"content" binding:"required"`
	ConversationID uint64 `json:"conversationId" binding:"required"`
	RequestID      string `json:"requestId" binding:"required"`
}

func chatHTTPError(c *gin.Context, err error) {
	code := http.StatusInternalServerError
	switch status.Code(err) {
	case codes.InvalidArgument:
		code = http.StatusBadRequest
	case codes.Unauthenticated:
		code = http.StatusUnauthorized
	case codes.PermissionDenied:
		code = http.StatusForbidden
	case codes.NotFound:
		code = http.StatusNotFound
	case codes.AlreadyExists, codes.Aborted:
		code = http.StatusConflict
	case codes.ResourceExhausted:
		code = http.StatusTooManyRequests
	case codes.DeadlineExceeded:
		code = http.StatusGatewayTimeout
	case codes.Unavailable:
		code = http.StatusServiceUnavailable
	case codes.Canceled:
		code = http.StatusRequestTimeout
	}
	message := status.Convert(err).Message()
	if code == http.StatusInternalServerError {
		message = "对话服务暂时不可用，请稍后重试"
	}
	response.Error(c, code, code, message)
}

func chatQueryID(c *gin.Context, name string, required bool) (uint64, bool) {
	value := c.Query(name)
	if value == "" && !required {
		return 0, true
	}
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "invalid "+name)
		return 0, false
	}
	return id, true
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32*1024)
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	hrID := middleware.GetUserID(c)
	resp, err := h.chatClient.SendMessage(c.Request.Context(), &pb.SendMessageRequest{
		HrId:           hrID,
		Content:        req.Content,
		ConversationId: req.ConversationID,
		RequestId:      req.RequestID,
	})
	if err != nil {
		chatHTTPError(c, err)
		return
	}

	response.Success(c, gin.H{"reply": resp.Reply})
}

func (h *ChatHandler) GetChatHistory(c *gin.Context) {
	conversationID, ok := chatQueryID(c, "conversationId", true)
	if !ok {
		return
	}
	beforeID, ok := chatQueryID(c, "beforeId", false)
	if !ok {
		return
	}
	limit := int64(40)
	if raw := c.Query("limit"); raw != "" {
		var err error
		limit, err = strconv.ParseInt(raw, 10, 32)
		if err != nil || limit < 1 || limit > 100 {
			response.BadRequest(c, "limit must be 1..100")
			return
		}
	}
	hrID := middleware.GetUserID(c)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	resp, err := h.chatClient.GetChatHistory(ctx, &pb.GetChatHistoryRequest{
		HrId:           hrID,
		ConversationId: conversationID, BeforeId: beforeID, Limit: int32(limit),
	})
	if err != nil {
		chatHTTPError(c, err)
		return
	}

	response.Success(c, gin.H{"messages": resp.Messages, "nextBeforeId": resp.NextBeforeId})
}

func (h *ChatHandler) CreateConversation(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	var req struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid conversation title")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	resp, err := h.chatClient.CreateConversation(ctx, &pb.CreateConversationRequest{HrId: middleware.GetUserID(c), Title: req.Title})
	if err != nil {
		chatHTTPError(c, err)
		return
	}
	response.Success(c, resp)
}

func (h *ChatHandler) ListConversations(c *gin.Context) {
	beforeID, ok := chatQueryID(c, "beforeId", false)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	resp, err := h.chatClient.ListConversations(ctx, &pb.ListConversationsRequest{HrId: middleware.GetUserID(c), BeforeId: beforeID})
	if err != nil {
		chatHTTPError(c, err)
		return
	}
	response.Success(c, gin.H{"conversations": resp.Conversations, "nextBeforeId": resp.NextBeforeId})
}

func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "invalid conversation ID")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	_, err = h.chatClient.DeleteConversation(ctx, &pb.DeleteConversationRequest{HrId: middleware.GetUserID(c), ConversationId: id})
	if err != nil {
		chatHTTPError(c, err)
		return
	}
	response.Success(c, nil)
}
