package handler

import (
	"web-gin-service/middleware"
	"web-gin-service/pb"
	"web-gin-service/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authClient pb.AuthServiceClient
}

func NewAuthHandler(client pb.AuthServiceClient) *AuthHandler {
	return &AuthHandler{authClient: client}
}

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) HrRegister(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	resp, err := h.authClient.HrRegister(c.Request.Context(), &pb.HrRegisterRequest{
		Username: req.Username,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"id": resp.Id})
}

func (h *AuthHandler) HrLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	resp, err := h.authClient.HrLogin(c.Request.Context(), &pb.HrLoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	token, err := middleware.GenerateToken(resp.Id, resp.Username, "hr")
	if err != nil {
		response.InternalError(c, "failed to generate token")
		return
	}

	response.Success(c, gin.H{
		"token":    token,
		"id":       resp.Id,
		"username": resp.Username,
		"name":     resp.Name,
	})
}

func (h *AuthHandler) CandidateRegister(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	resp, err := h.authClient.CandidateRegister(c.Request.Context(), &pb.CandidateRegisterRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"id": resp.Id})
}

func (h *AuthHandler) CandidateLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	resp, err := h.authClient.CandidateLogin(c.Request.Context(), &pb.CandidateLoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	token, err := middleware.GenerateToken(resp.Id, resp.Username, "candidate")
	if err != nil {
		response.InternalError(c, "failed to generate token")
		return
	}

	response.Success(c, gin.H{
		"token":    token,
		"id":       resp.Id,
		"username": resp.Username,
	})
}
