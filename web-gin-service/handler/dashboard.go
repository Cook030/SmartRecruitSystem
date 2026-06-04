package handler

import (
	"strconv"

	"web-gin-service/middleware"
	"web-gin-service/pb"
	"web-gin-service/response"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	dashboardClient pb.DashboardServiceClient
}

func NewDashboardHandler(client pb.DashboardServiceClient) *DashboardHandler {
	return &DashboardHandler{dashboardClient: client}
}

func (h *DashboardHandler) GetOverview(c *gin.Context) {
	hrID := middleware.GetUserID(c)
	resp, err := h.dashboardClient.GetOverview(c.Request.Context(), &pb.DashboardOverviewRequest{HrId: hrID})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, resp)
}

func (h *DashboardHandler) GetJobTrend(c *gin.Context) {
	h.getTrend(c, true)
}

func (h *DashboardHandler) GetApplicationTrend(c *gin.Context) {
	h.getTrend(c, false)
}

func (h *DashboardHandler) getTrend(c *gin.Context, jobTrend bool) {
	hrID := middleware.GetUserID(c)
	req := &pb.DashboardTrendRequest{
		HrId:      hrID,
		StartDate: c.Query("startDate"),
		EndDate:   c.Query("endDate"),
		Range:     c.DefaultQuery("range", "week"),
	}
	var (
		resp *pb.DashboardTrendResponse
		err  error
	)
	if jobTrend {
		resp, err = h.dashboardClient.GetJobTrend(c.Request.Context(), req)
	} else {
		resp, err = h.dashboardClient.GetApplicationTrend(c.Request.Context(), req)
	}
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, resp)
}

func (h *DashboardHandler) GetHotJobs(c *gin.Context) {
	hrID := middleware.GetUserID(c)
	limit := parseLimit(c.DefaultQuery("limit", "10"))
	resp, err := h.dashboardClient.GetHotJobs(c.Request.Context(), &pb.DashboardHotJobsRequest{HrId: hrID, Limit: limit})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, resp)
}

func (h *DashboardHandler) GetHotSkills(c *gin.Context) {
	hrID := middleware.GetUserID(c)
	limit := parseLimit(c.DefaultQuery("limit", "10"))
	resp, err := h.dashboardClient.GetHotSkills(c.Request.Context(), &pb.DashboardHotSkillsRequest{HrId: hrID, Limit: limit})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, resp)
}

func (h *DashboardHandler) GetCandidateFunnel(c *gin.Context) {
	hrID := middleware.GetUserID(c)
	resp, err := h.dashboardClient.GetCandidateFunnel(c.Request.Context(), &pb.DashboardFunnelRequest{HrId: hrID})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, resp)
}

func (h *DashboardHandler) GetAIMatchStats(c *gin.Context) {
	hrID := middleware.GetUserID(c)
	resp, err := h.dashboardClient.GetAIMatchStats(c.Request.Context(), &pb.DashboardAIMatchRequest{HrId: hrID})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, resp)
}

func parseLimit(raw string) int32 {
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 10
	}
	if limit > 100 {
		return 100
	}
	return int32(limit)
}
