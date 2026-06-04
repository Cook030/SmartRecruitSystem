package handler

import (
	"strconv"

	"web-gin-service/middleware"
	"web-gin-service/pb"
	"web-gin-service/response"

	"github.com/gin-gonic/gin"
)

type JobHandler struct {
	jobClient       pb.JobServiceClient
	candidateClient pb.CandidateServiceClient
}

func NewJobHandler(jobClient pb.JobServiceClient, candidateClient pb.CandidateServiceClient) *JobHandler {
	return &JobHandler{jobClient: jobClient, candidateClient: candidateClient}
}

type createJobRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Requirement string `json:"requirement"`
	Salary      string `json:"salary"`
	Location    string `json:"location"`
}

type updateJobRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Requirement string `json:"requirement"`
	Salary      string `json:"salary"`
	Location    string `json:"location"`
}

func (h *JobHandler) CreateJob(c *gin.Context) {
	var req createJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	hrID := middleware.GetUserID(c)
	resp, err := h.jobClient.CreateJob(c.Request.Context(), &pb.CreateJobRequest{
		HrId:        hrID,
		Title:       req.Title,
		Description: req.Description,
		Requirement: req.Requirement,
		Salary:      req.Salary,
		Location:    req.Location,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"id": resp.Id})
}

func (h *JobHandler) UpdateJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid job id")
		return
	}

	var req updateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	hrID := middleware.GetUserID(c)
	_, err = h.jobClient.UpdateJob(c.Request.Context(), &pb.UpdateJobRequest{
		Id:          id,
		HrId:       hrID,
		Title:       req.Title,
		Description: req.Description,
		Requirement: req.Requirement,
		Salary:      req.Salary,
		Location:    req.Location,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *JobHandler) OfflineJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid job id")
		return
	}

	hrID := middleware.GetUserID(c)
	_, err = h.jobClient.OfflineJob(c.Request.Context(), &pb.OfflineJobRequest{
		Id:   id,
		HrId: hrID,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *JobHandler) ListHrJobs(c *gin.Context) {
	hrID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	resp, err := h.jobClient.ListHrJobs(c.Request.Context(), &pb.ListHrJobsRequest{
		HrId:     hrID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"jobs": resp.Jobs, "total": resp.Total})
}

func (h *JobHandler) ListPublicJobs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	resp, err := h.jobClient.ListPublicJobs(c.Request.Context(), &pb.ListPublicJobsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"jobs": resp.Jobs, "total": resp.Total})
}

func (h *JobHandler) ListJobApplications(c *gin.Context) {
	idStr := c.Param("id")
	jobID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid job id")
		return
	}

	hrID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	resp, err := h.candidateClient.ListJobApplications(c.Request.Context(), &pb.ListJobApplicationsRequest{
		JobId:    jobID,
		HrId:     hrID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"applications": resp.Applications, "total": resp.Total})
}

func (h *JobHandler) GetApplicationDetail(c *gin.Context) {
	idStr := c.Param("id")
	appID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	hrID := middleware.GetUserID(c)
	resp, err := h.candidateClient.GetApplicationDetail(c.Request.Context(), &pb.GetApplicationDetailRequest{
		ApplicationId: appID,
		HrId:          hrID,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"application": resp.Application,
		"profile":     resp.Profile,
		"resume":      resp.Resume,
	})
}
