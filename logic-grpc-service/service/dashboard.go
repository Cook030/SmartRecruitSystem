package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"logic-grpc-service/pb"
	"logic-grpc-service/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DashboardService struct {
	pb.UnimplementedDashboardServiceServer
	repo *repository.Repository
}

func NewDashboardService(repo *repository.Repository) *DashboardService {
	return &DashboardService{repo: repo}
}

func (s *DashboardService) invalidateDashboardCache(ctx context.Context, hrID uint64) {
	s.repo.CacheDelByPrefix(ctx, fmt.Sprintf("dashboard:overview:%d", hrID))
	s.repo.CacheDelByPrefix(ctx, fmt.Sprintf("dashboard:hot_jobs:%d", hrID))
	s.repo.CacheDelByPrefix(ctx, fmt.Sprintf("dashboard:hot_skills:%d", hrID))
	s.repo.CacheDelByPrefix(ctx, fmt.Sprintf("dashboard:funnel:%d", hrID))
	s.repo.CacheDelByPrefix(ctx, fmt.Sprintf("dashboard:ai_match:%d", hrID))
	s.repo.CacheDelByPrefix(ctx, fmt.Sprintf("dashboard:job_trend:%d", hrID))
	s.repo.CacheDelByPrefix(ctx, fmt.Sprintf("dashboard:app_trend:%d", hrID))
}

func (s *DashboardService) GetOverview(ctx context.Context, req *pb.DashboardOverviewRequest) (*pb.DashboardOverviewResponse, error) {
	key := fmt.Sprintf("dashboard:overview:%d", req.HrId)
	var resp pb.DashboardOverviewResponse
	err := s.repo.CacheGet(ctx, key, &resp, 2*time.Minute, func() (interface{}, error) {
		totalJobs, err := s.repo.CountJobsByHrID(req.HrId)
		if err != nil { return nil, err }
		onlineJobs, err := s.repo.CountOnlineJobsByHrID(req.HrId)
		if err != nil { return nil, err }
		offlineJobs, err := s.repo.CountOfflineJobsByHrID(req.HrId)
		if err != nil { return nil, err }
		totalApplications, err := s.repo.CountApplicationsByHrID(req.HrId)
		if err != nil { return nil, err }
		totalCandidates, err := s.repo.CountCandidatesByHrID(req.HrId)
		if err != nil { return nil, err }
		completeProfiles, err := s.repo.CountCompleteProfilesByHrID(req.HrId)
		if err != nil { return nil, err }
		totalResumes, err := s.repo.CountResumesUploadedByHrID(req.HrId)
		if err != nil { return nil, err }
		totalChats, err := s.repo.CountChatsByHrID(req.HrId)
		if err != nil { return nil, err }
		return pb.DashboardOverviewResponse{TotalJobs: totalJobs, OnlineJobs: onlineJobs, OfflineJobs: offlineJobs, TotalApplications: totalApplications, TotalCandidates: totalCandidates, CompleteProfiles: completeProfiles, TotalResumes: totalResumes, TotalChats: totalChats}, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get overview")
	}
	return &resp, nil
}

func (s *DashboardService) GetHotJobs(ctx context.Context, req *pb.DashboardHotJobsRequest) (*pb.DashboardHotJobsResponse, error) {
	key := fmt.Sprintf("dashboard:hot_jobs:%d:limit:%d", req.HrId, req.Limit)
	var resp pb.DashboardHotJobsResponse
	err := s.repo.CacheGet(ctx, key, &resp, 5*time.Minute, func() (interface{}, error) {
		items, err := s.repo.GetHotJobsByHrID(req.HrId)
		if err != nil { return nil, err }
		limit := int(req.Limit)
		if limit <= 0 || limit > len(items) { limit = len(items) }
		jobs := make([]*pb.HotJobItem, 0, limit)
		for i := 0; i < limit; i++ { jobs = append(jobs, &pb.HotJobItem{JobId: items[i].JobID, Title: items[i].Title, Location: items[i].Location, Salary: items[i].Salary, ApplicationCount: items[i].ApplicationCount}) }
		return pb.DashboardHotJobsResponse{Jobs: jobs}, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get hot jobs")
	}
	return &resp, nil
}

func (s *DashboardService) GetHotSkills(ctx context.Context, req *pb.DashboardHotSkillsRequest) (*pb.DashboardHotSkillsResponse, error) {
	key := fmt.Sprintf("dashboard:hot_skills:%d:limit:%d", req.HrId, req.Limit)
	var resp pb.DashboardHotSkillsResponse
	err := s.repo.CacheGet(ctx, key, &resp, 5*time.Minute, func() (interface{}, error) {
		profiles, err := s.repo.GetAllCandidateProfilesWithSkillsByHrID(req.HrId)
		if err != nil { return nil, err }
		counts := map[string]int64{}
		for _, p := range profiles {
			for _, skill := range strings.Split(p.Skills, ",") {
				skill = strings.TrimSpace(skill)
				if skill != "" { counts[skill]++ }
			}
		}
		items := make([]*pb.HotSkillItem, 0, len(counts))
		for skill, count := range counts { items = append(items, &pb.HotSkillItem{Skill: skill, Count: count}) }
		sort.Slice(items, func(i, j int) bool { if items[i].Count == items[j].Count { return items[i].Skill < items[j].Skill }; return items[i].Count > items[j].Count })
		limit := int(req.Limit)
		if limit <= 0 || limit > len(items) { limit = len(items) }
		return pb.DashboardHotSkillsResponse{Skills: items[:limit]}, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get hot skills")
	}
	return &resp, nil
}

func (s *DashboardService) GetCandidateFunnel(ctx context.Context, req *pb.DashboardFunnelRequest) (*pb.DashboardFunnelResponse, error) {
	key := fmt.Sprintf("dashboard:funnel:%d", req.HrId)
	var resp pb.DashboardFunnelResponse
	err := s.repo.CacheGet(ctx, key, &resp, 2*time.Minute, func() (interface{}, error) {
		registered, _ := s.repo.CountCandidatesByHrID(req.HrId)
		profileComplete, _ := s.repo.CountCompleteProfilesByHrID(req.HrId)
		resumeUploaded, _ := s.repo.CountResumesUploadedByHrID(req.HrId)
		applied, _ := s.repo.CountApplicationsByHrID(req.HrId)
		return pb.DashboardFunnelResponse{Registered: registered, ProfileComplete: profileComplete, ResumeUploaded: resumeUploaded, Applied: applied}, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get funnel")
	}
	return &resp, nil
}

func (s *DashboardService) GetAIMatchStats(ctx context.Context, req *pb.DashboardAIMatchRequest) (*pb.DashboardAIMatchResponse, error) {
	key := fmt.Sprintf("dashboard:ai_match:%d", req.HrId)
	var resp pb.DashboardAIMatchResponse
	err := s.repo.CacheGet(ctx, key, &resp, 2*time.Minute, func() (interface{}, error) {
		totalChats, hrMessages, aiMessages, err := s.repo.CountChatMessagesByHrID(req.HrId)
		if err != nil { return nil, err }
		matchTasks, successTasks, failedTasks, avgScore, err := s.repo.CountAIMatchResultsByHrID(req.HrId)
		if err != nil { return nil, err }
		return pb.DashboardAIMatchResponse{TotalChats: totalChats, HrMessages: hrMessages, AiMessages: aiMessages, MatchTasks: matchTasks, SuccessTasks: successTasks, FailedTasks: failedTasks, AverageScore: avgScore}, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get ai stats")
	}
	return &resp, nil
}

func (s *DashboardService) GetJobTrend(ctx context.Context, req *pb.DashboardTrendRequest) (*pb.DashboardTrendResponse, error) {
	return s.getTrend(req.HrId, req.StartDate, req.EndDate, req.Range, true)
}

func (s *DashboardService) GetApplicationTrend(ctx context.Context, req *pb.DashboardTrendRequest) (*pb.DashboardTrendResponse, error) {
	return s.getTrend(req.HrId, req.StartDate, req.EndDate, req.Range, false)
}

func (s *DashboardService) getTrend(hrID uint64, startDate, endDate, rng string, jobs bool) (*pb.DashboardTrendResponse, error) {
	ctx := context.Background()
	prefix := "job_trend"
	if !jobs { prefix = "app_trend" }
	key := fmt.Sprintf("dashboard:%s:%d:%s:%s:%s", prefix, hrID, startDate, endDate, rng)
	var resp pb.DashboardTrendResponse
	err := s.repo.CacheGet(ctx, key, &resp, 2*time.Minute, func() (interface{}, error) {
		start, end := parseRange(startDate, endDate, rng)
		points := map[string]int64{}
		if jobs {
			items, err := s.repo.GetJobsCreatedBetween(hrID, start, end)
			if err != nil { return nil, err }
			for _, item := range items { points[item.Date] += item.Count }
		} else {
			items, err := s.repo.GetApplicationsCreatedBetween(hrID, start, end)
			if err != nil { return nil, err }
			for _, item := range items { points[item.Date] += item.Count }
		}
		out := make([]*pb.TrendPoint, 0, len(points))
		for date, count := range points { out = append(out, &pb.TrendPoint{Date: date, Count: count}) }
		sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
		return pb.DashboardTrendResponse{Points: out}, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get trend")
	}
	return &resp, nil
}

func parseRange(startDate, endDate, rng string) (time.Time, time.Time) {
	layout := "2006-01-02"
	if startDate != "" && endDate != "" {
		start, _ := time.ParseInLocation(layout, startDate, time.Local)
		end, _ := time.ParseInLocation(layout, endDate, time.Local)
		return start, end.Add(23 * time.Hour + 59 * time.Minute + 59 * time.Second)
	}
	now := time.Now()
	switch rng {
	case "year": return now.AddDate(-1, 0, 0), now
	case "month": return now.AddDate(0, -1, 0), now
	default: return now.AddDate(0, 0, -7), now
	}
}

