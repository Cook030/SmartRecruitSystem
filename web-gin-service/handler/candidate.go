package handler

import (
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"web-gin-service/middleware"
	"web-gin-service/pb"
	"web-gin-service/response"

	"github.com/gin-gonic/gin"
)

type CandidateHandler struct {
	candidateClient pb.CandidateServiceClient
}

func NewCandidateHandler(client pb.CandidateServiceClient) *CandidateHandler {
	return &CandidateHandler{candidateClient: client}
}

type updateProfileRequest struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Education  string `json:"education"`
	University string `json:"university"`
	Experience string `json:"experience"`
	Skills     string `json:"skills"`
}

func (h *CandidateHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	resp, err := h.candidateClient.GetProfile(c.Request.Context(), &pb.GetProfileRequest{
		CandidateId: userID,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, resp.Profile)
}

func (h *CandidateHandler) UpdateProfile(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	resp, err := h.candidateClient.UpdateProfile(c.Request.Context(), &pb.UpdateProfileRequest{
		CandidateId: userID,
		Name:        req.Name,
		Phone:       req.Phone,
		Education:   req.Education,
		University:  req.University,
		Experience:  req.Experience,
		Skills:      req.Skills,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"isComplete": resp.IsComplete})
}

func (h *CandidateHandler) UploadResume(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	defer file.Close()

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(header.Filename), "."))
	allowedExts := map[string]bool{"pdf": true, "doc": true, "docx": true}
	if !allowedExts[ext] {
		response.BadRequest(c, "only pdf/doc/docx files are allowed")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		response.InternalError(c, "failed to read file")
		return
	}

	userID := middleware.GetUserID(c)
	resp, err := h.candidateClient.UploadResume(c.Request.Context(), &pb.UploadResumeRequest{
		CandidateId: userID,
		FileName:    header.Filename,
		FileData:    data,
		FileSize:    int64(len(data)),
		FileType:    ext,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"id": resp.Id, "ossKey": resp.OssKey})
}

func (h *CandidateHandler) GetResume(c *gin.Context) {
	userID := middleware.GetUserID(c)
	resp, err := h.candidateClient.GetResume(c.Request.Context(), &pb.GetResumeRequest{
		CandidateId: userID,
	})
	if err != nil {
		response.NotFound(c, "resume not found")
		return
	}
	response.Success(c, resp.Resume)
}

func (h *CandidateHandler) DeleteResume(c *gin.Context) {
	userID := middleware.GetUserID(c)
	_, err := h.candidateClient.DeleteResume(c.Request.Context(), &pb.DeleteResumeRequest{
		CandidateId: userID,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

func (h *CandidateHandler) ApplyJob(c *gin.Context) {
	jobIDStr := c.Param("jobId")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid job id")
		return
	}

	userID := middleware.GetUserID(c)
	resp, err := h.candidateClient.ApplyJob(c.Request.Context(), &pb.ApplyJobRequest{
		CandidateId: userID,
		JobId:       jobID,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"id": resp.Id})
}

func (h *CandidateHandler) ListMyApplications(c *gin.Context) {
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	resp, err := h.candidateClient.ListMyApplications(c.Request.Context(), &pb.ListMyApplicationsRequest{
		CandidateId: userID,
		Page:        int32(page),
		PageSize:    int32(pageSize),
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"applications": resp.Applications, "total": resp.Total})
}
