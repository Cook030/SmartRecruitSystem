package handler

import (
	"web-gin-service/middleware"
	"web-gin-service/pb"
	"web-gin-service/response"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	chatClient pb.ChatServiceClient
}

func NewChatHandler(client pb.ChatServiceClient) *ChatHandler {
	return &ChatHandler{chatClient: client}
}

type sendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	hrID := middleware.GetUserID(c)
	resp, err := h.chatClient.SendMessage(c.Request.Context(), &pb.SendMessageRequest{
		HrId:    hrID,
		Content: req.Content,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"reply": resp.Reply})
}

func (h *ChatHandler) GetChatHistory(c *gin.Context) {
	hrID := middleware.GetUserID(c)
	resp, err := h.chatClient.GetChatHistory(c.Request.Context(), &pb.GetChatHistoryRequest{
		HrId: hrID,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"messages": resp.Messages})
}
