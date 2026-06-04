package handler

import (
	"strconv"

	"web-gin-service/middleware"
	"web-gin-service/pb"
	"web-gin-service/response"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	taskClient pb.TaskServiceClient
}

func NewTaskHandler(client pb.TaskServiceClient) *TaskHandler {
	return &TaskHandler{taskClient: client}
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid task id")
		return
	}

	resp, err := h.taskClient.GetTask(c.Request.Context(), &pb.GetTaskRequest{TaskId: taskID})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, resp)
}

func (h *TaskHandler) TriggerAIMatch(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	hrID := middleware.GetUserID(c)
	resp, err := h.taskClient.TriggerAIMatch(c.Request.Context(), &pb.TriggerAIMatchRequest{
		ApplicationId: appID,
		HrId:          hrID,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, resp)
}

func (h *TaskHandler) TriggerResumeParse(c *gin.Context) {
	resumeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid resume id")
		return
	}

	candidateID := middleware.GetUserID(c)
	resp, err := h.taskClient.TriggerResumeParse(c.Request.Context(), &pb.TriggerResumeParseRequest{
		ResumeId:    resumeID,
		CandidateId: candidateID,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, resp)
}

func (h *TaskHandler) RefreshDashboard(c *gin.Context) {
	hrID := middleware.GetUserID(c)
	resp, err := h.taskClient.RefreshDashboard(c.Request.Context(), &pb.RefreshDashboardRequest{
		HrId: hrID,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, resp)
}

func (h *TaskHandler) GetResumeParseResult(c *gin.Context) {
	resumeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid resume id")
		return
	}

	resp, err := h.taskClient.GetResumeParseResult(c.Request.Context(), &pb.GetResumeParseResultRequest{
		ResumeId: resumeID,
	})
	if err != nil {
		response.NotFound(c, "parse result not found")
		return
	}
	response.Success(c, resp)
}

func (h *TaskHandler) DeleteResumeParseResult(c *gin.Context) {
	resumeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid resume id")
		return
	}

	_, err = h.taskClient.DeleteResumeParseResult(c.Request.Context(), &pb.DeleteResumeParseResultRequest{
		ResumeId: resumeID,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, nil)
}
