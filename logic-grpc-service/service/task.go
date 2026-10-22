package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"logic-grpc-service/ai"
	"logic-grpc-service/config"
	"logic-grpc-service/model"
	"logic-grpc-service/oss"
	"logic-grpc-service/pb"
	"logic-grpc-service/repository"
	"logic-grpc-service/telemetry"

	"github.com/ledongthuc/pdf"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TaskService struct {
	pb.UnimplementedTaskServiceServer
	repo        *repository.Repository
	cfg         *config.Config
	workerCtx   context.Context
	stopWorkers context.CancelFunc
	workers     *sync.WaitGroup
}

func NewTaskService(ctx context.Context, repo *repository.Repository, cfg *config.Config) *TaskService {
	ctx, cancel := context.WithCancel(ctx)
	ts := &TaskService{repo: repo, cfg: cfg, workerCtx: ctx, stopWorkers: cancel, workers: &sync.WaitGroup{}}
	ts.StartWorker()
	return ts
}

func (s *TaskService) Close() {
	s.stopWorkers()
	done := make(chan struct{})
	go func() { s.workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}

func (s *TaskService) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	task := &model.AsyncTask{
		TaskType:   req.TaskType,
		BusinessID: req.BusinessId,
		UserID:     req.UserId,
		UserRole:   req.UserRole,
		Params:     req.Params,
		Status:     "pending",
		MaxRetries: 3,
	}
	if err := s.repo.CreateAndQueueTask(ctx, task); err != nil {
		return nil, status.Error(codes.Internal, "failed to create task")
	}
	return &pb.CreateTaskResponse{TaskId: task.ID}, nil
}

func (s *TaskService) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	task, err := s.repo.GetAsyncTaskByID(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "task not found")
	}
	resp := &pb.GetTaskResponse{
		Id:           task.ID,
		TaskType:     task.TaskType,
		BusinessId:   task.BusinessID,
		UserId:       task.UserID,
		UserRole:     task.UserRole,
		Status:       task.Status,
		Params:       task.Params,
		Result:       task.Result,
		ErrorMessage: task.ErrorMessage,
		RetryCount:   int32(task.RetryCount),
		MaxRetries:   int32(task.MaxRetries),
		CreatedAt:    task.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if task.StartedAt != nil {
		resp.StartedAt = task.StartedAt.Format("2006-01-02 15:04:05")
	}
	if task.CompletedAt != nil {
		resp.CompletedAt = task.CompletedAt.Format("2006-01-02 15:04:05")
	}
	return resp, nil
}

func (s *TaskService) TriggerAIMatch(ctx context.Context, req *pb.TriggerAIMatchRequest) (*pb.CreateTaskResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	params, _ := json.Marshal(map[string]uint64{
		"application_id": req.ApplicationId,
		"hr_id":          req.HrId,
	})
	task := &model.AsyncTask{
		TaskType:   "ai_match",
		BusinessID: req.ApplicationId,
		UserID:     req.HrId,
		UserRole:   "hr",
		Params:     string(params),
		Status:     "pending",
		MaxRetries: 3,
	}
	if err := s.repo.CreateAndQueueTask(ctx, task); err != nil {
		return nil, status.Error(codes.Internal, "failed to create task")
	}
	return &pb.CreateTaskResponse{TaskId: task.ID}, nil
}

func (s *TaskService) TriggerResumeParse(ctx context.Context, req *pb.TriggerResumeParseRequest) (*pb.CreateTaskResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	params, _ := json.Marshal(map[string]uint64{
		"resume_id":    req.ResumeId,
		"candidate_id": req.CandidateId,
	})
	task := &model.AsyncTask{
		TaskType:   "resume_parse",
		BusinessID: req.ResumeId,
		UserID:     req.CandidateId,
		UserRole:   "candidate",
		Params:     string(params),
		Status:     "pending",
		MaxRetries: 3,
	}
	if err := s.repo.CreateAndQueueTask(ctx, task); err != nil {
		return nil, status.Error(codes.Internal, "failed to create task")
	}
	return &pb.CreateTaskResponse{TaskId: task.ID}, nil
}

func (s *TaskService) RefreshDashboard(ctx context.Context, req *pb.RefreshDashboardRequest) (*pb.CreateTaskResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	params, _ := json.Marshal(map[string]uint64{
		"hr_id": req.HrId,
	})
	task := &model.AsyncTask{
		TaskType:   "stats_refresh",
		BusinessID: req.HrId,
		UserID:     req.HrId,
		UserRole:   "hr",
		Params:     string(params),
		Status:     "pending",
		MaxRetries: 3,
	}
	if err := s.repo.CreateAndQueueTask(ctx, task); err != nil {
		return nil, status.Error(codes.Internal, "failed to create task")
	}
	return &pb.CreateTaskResponse{TaskId: task.ID}, nil
}

func (s *TaskService) GetResumeParseResult(ctx context.Context, req *pb.GetResumeParseResultRequest) (*pb.GetResumeParseResultResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	result, err := s.repo.GetResumeParseResultByResumeID(req.ResumeId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "parse result not found")
	}
	return &pb.GetResumeParseResultResponse{
		Id:          result.ID,
		ResumeId:    result.ResumeID,
		CandidateId: result.CandidateID,
		Education:   result.Education,
		Experience:  result.Experience,
		Projects:    result.Projects,
		Skills:      result.Skills,
		RawResult:   result.RawResult,
		CreatedAt:   result.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   result.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *TaskService) DeleteResumeParseResult(ctx context.Context, req *pb.DeleteResumeParseResultRequest) (*pb.DeleteResumeParseResultResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	if err := s.repo.DeleteResumeParseResultByResumeID(req.ResumeId); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete parse result")
	}
	return &pb.DeleteResumeParseResultResponse{}, nil
}

func (s *TaskService) StartWorker() {
	log.Println("Task worker started")
	if s.repo.Redis.Available() {
		s.workers.Add(1)
		go s.redisWorker()
	}
	s.workers.Add(1)
	go s.mysqlWorker()
}

func (s *TaskService) redisWorker() {
	defer s.workers.Done()
	for s.workerCtx.Err() == nil {
		result, err := s.repo.Redis.BRPop(s.workerCtx, 5*time.Second, "task:queue")
		if err != nil || result == "" {
			continue
		}
		var taskID uint64
		if _, err := fmt.Sscanf(result, "%d", &taskID); err != nil {
			continue
		}
		task, err := s.repo.WithContext(s.workerCtx).GetAsyncTaskByID(taskID)
		if err != nil {
			continue
		}
		if task.Status == "pending" || task.Status == "retrying" {
			s.processTask(task)
		}
	}
}

func (s *TaskService) mysqlWorker() {
	defer s.workers.Done()
	for s.workerCtx.Err() == nil {
		tasks, err := s.repo.WithContext(s.workerCtx).ListPendingTasks(10)
		if err != nil {
			if !workerDelay(s.workerCtx, 5*time.Second) {
				return
			}
			continue
		}
		for _, t := range tasks {
			s.processTask(&t)
		}
		if len(tasks) == 0 {
			if !workerDelay(s.workerCtx, 10*time.Second) {
				return
			}
		} else {
			if !workerDelay(s.workerCtx, 2*time.Second) {
				return
			}
		}
	}
}

func workerDelay(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (s *TaskService) processTask(task *model.AsyncTask) {
	if s.workerCtx.Err() != nil {
		return
	}
	parent := otel.GetTextMapPropagator().Extract(s.workerCtx, propagation.MapCarrier{"traceparent": task.TraceParent, "tracestate": task.TraceState})
	taskType := telemetry.TaskType(task.TaskType)
	ctx, end := telemetry.Start(parent, "task.execute", attribute.String("task_type", taskType))
	trace.SpanFromContext(ctx).SetAttributes(attribute.Int64("task.id", int64(task.ID)), attribute.Int("task.retry", task.RetryCount))
	outcome := "error"
	defer func() { end(outcome) }()
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	if task.RetryCount == 0 {
		telemetry.TaskLatency(ctx, taskType, "first_start", time.Since(task.CreatedAt))
	}
	now := time.Now()
	task.Status = "running"
	task.StartedAt = &now
	if err := s.repo.UpdateAsyncTask(ctx, task); err != nil {
		outcome = "state_update_failed"
		return
	}

	var taskErr error
	switch task.TaskType {
	case "resume_parse":
		taskErr = s.processResumeParse(ctx, task)
	case "ai_match":
		taskErr = s.processAIMatch(ctx, task)
	case "stats_refresh":
		taskErr = s.processStatsRefresh(ctx, task)
	case "notification":
		taskErr = s.processNotification(ctx, task)
	default:
		taskErr = fmt.Errorf("unknown task type: %s", task.TaskType)
	}

	completedAt := time.Now()
	task.CompletedAt = &completedAt

	if taskErr != nil {
		task.ErrorMessage = taskErr.Error()
		if task.RetryCount < task.MaxRetries {
			task.Status = "retrying"
			task.RetryCount++
		} else {
			task.Status = "failed"
		}
		slog.WarnContext(ctx, "task attempt failed", "task_id", task.ID, "task_type", taskType, "outcome", telemetry.Outcome(taskErr), "retries", task.RetryCount)
	} else {
		task.Status = "success"
		slog.InfoContext(ctx, "task completed", "task_id", task.ID, "task_type", taskType)
	}
	// A shutdown cancels dependency calls but must still persist the retry state.
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if err := s.repo.UpdateAsyncTask(cleanup, task); err != nil {
		outcome = "state_update_failed"
		return
	}
	outcome = telemetry.Outcome(taskErr)
	telemetry.Event(ctx, "task."+task.Status, attribute.String("task_type", taskType))
	if task.Status == "success" || task.Status == "failed" {
		telemetry.TaskLatency(ctx, taskType, "end_to_end", time.Since(task.CreatedAt))
	}
}

func (s *TaskService) processResumeParse(ctx context.Context, task *model.AsyncTask) error {
	var params struct {
		ResumeID    uint64 `json:"resume_id"`
		CandidateID uint64 `json:"candidate_id"`
		OssKey      string `json:"oss_key"`
	}
	if err := json.Unmarshal([]byte(task.Params), &params); err != nil {
		return err
	}

	resume, err := s.repo.GetResumeByID(params.ResumeID)
	if err != nil {
		return fmt.Errorf("resume not found: %v", err)
	}

	fileData, err := oss.DownloadFile(ctx, s.cfg, resume.OssKey)
	if err != nil {
		return fmt.Errorf("failed to download resume from OSS: %w", err)
	}

	var textContent string
	switch resume.FileType {
	case "pdf":
		r, err := pdf.NewReader(bytes.NewReader(fileData), int64(len(fileData)))
		if err != nil {
			return fmt.Errorf("failed to read PDF: %v", err)
		}
		var buf strings.Builder
		for i := 1; i <= r.NumPage(); i++ {
			page := r.Page(i)
			if page.V.IsNull() {
				continue
			}
			text, err := page.GetPlainText(nil)
			if err != nil {
				continue
			}
			buf.WriteString(text)
			buf.WriteString("\n")
		}
		textContent = buf.String()
	default:
		textContent = string(fileData)
	}

	if textContent == "" {
		return fmt.Errorf("failed to extract text from resume")
	}

	prompt := fmt.Sprintf("请解析以下简历信息，提取教育经历、工作经历、项目经历和技能标签，以JSON格式返回：{\"education\":\"\",\"experience\":\"\",\"projects\":\"\",\"skills\":\"\"}\n\n简历内容：%s", textContent)

	aiResp, err := ai.CallAI(ctx, s.cfg, prompt)
	if err != nil {
		return fmt.Errorf("AI call failed: %w", err)
	}

	result := &model.ResumeParseResult{
		ResumeID:    params.ResumeID,
		CandidateID: params.CandidateID,
		RawResult:   aiResp,
	}

	var parsed struct {
		Education  string `json:"education"`
		Experience string `json:"experience"`
		Projects   string `json:"projects"`
		Skills     string `json:"skills"`
	}
	if err := json.Unmarshal([]byte(aiResp), &parsed); err == nil {
		result.Education = parsed.Education
		result.Experience = parsed.Experience
		result.Projects = parsed.Projects
		result.Skills = parsed.Skills
	} else {
		telemetry.Event(ctx, "ai.invalid_json", attribute.String("task_type", "resume_parse"))
	}

	if err := s.repo.SaveResumeParseResult(result); err != nil {
		return err
	}

	s.repo.CacheDel(ctx, fmt.Sprintf("ai:resume_parse:%d", params.ResumeID))

	parseResult := map[string]interface{}{
		"resume_id":  params.ResumeID,
		"parsed":     true,
		"education":  result.Education,
		"experience": result.Experience,
		"projects":   result.Projects,
		"skills":     result.Skills,
	}
	if data, err := json.Marshal(parseResult); err == nil {
		task.Result = string(data)
	} else {
		task.Result = fmt.Sprintf(`{"resume_id":%d,"parsed":true}`, params.ResumeID)
	}
	return nil
}

func (s *TaskService) processAIMatch(ctx context.Context, task *model.AsyncTask) error {
	var params struct {
		ApplicationID uint64 `json:"application_id"`
		HrID          uint64 `json:"hr_id"`
	}
	if err := json.Unmarshal([]byte(task.Params), &params); err != nil {
		return err
	}

	application, err := s.repo.GetApplicationByID(params.ApplicationID)
	if err != nil {
		return fmt.Errorf("application not found: %v", err)
	}

	job, err := s.repo.GetJobByID(application.JobID)
	if err != nil {
		return fmt.Errorf("job not found: %v", err)
	}

	profile, _ := s.repo.GetProfileByCandidateID(application.CandidateID)
	candidateInfo := ""
	if profile != nil {
		candidateInfo = fmt.Sprintf("姓名:%s, 学历:%s, 学校:%s, 技能:%s, 经验:%s",
			profile.Name, profile.Education, profile.University, profile.Skills, profile.Experience)
	}

	prompt := fmt.Sprintf(`请分析候选人与岗位的匹配度，以JSON格式返回：{"match_score":0,"match_reason":"","strengths":"","weaknesses":"","interview_tips":""}

岗位要求：%s
候选人信息：%s`, job.Requirement, candidateInfo)

	aiResp, err := ai.CallAI(ctx, s.cfg, prompt)
	if err != nil {
		return fmt.Errorf("AI call failed: %w", err)
	}

	result := &model.AIMatchResult{
		JobID:         job.ID,
		CandidateID:   application.CandidateID,
		ApplicationID: params.ApplicationID,
		RawResult:     aiResp,
	}

	var parsed struct {
		MatchScore    float64 `json:"match_score"`
		MatchReason   string  `json:"match_reason"`
		Strengths     string  `json:"strengths"`
		Weaknesses    string  `json:"weaknesses"`
		InterviewTips string  `json:"interview_tips"`
	}
	if err := json.Unmarshal([]byte(aiResp), &parsed); err == nil {
		result.MatchScore = parsed.MatchScore
		result.MatchReason = parsed.MatchReason
		result.Strengths = parsed.Strengths
		result.Weaknesses = parsed.Weaknesses
		result.InterviewTips = parsed.InterviewTips
	} else {
		telemetry.Event(ctx, "ai.invalid_json", attribute.String("task_type", "ai_match"))
	}
	if parsed.MatchScore < 0 || parsed.MatchScore > 100 {
		telemetry.Event(ctx, "ai.invalid_score")
	}

	if err := s.repo.SaveAIMatchResult(result); err != nil {
		return err
	}

	s.repo.CacheDel(ctx, fmt.Sprintf("ai:job_match:%d:%d", job.ID, application.CandidateID))
	task.Result = fmt.Sprintf(`{"application_id":%d,"match_score":%.1f}`, params.ApplicationID, result.MatchScore)
	return nil
}

func (s *TaskService) processStatsRefresh(ctx context.Context, task *model.AsyncTask) error {
	var params struct {
		HrID uint64 `json:"hr_id"`
	}
	if err := json.Unmarshal([]byte(task.Params), &params); err != nil {
		return err
	}

	stats := map[string]interface{}{}
	totalJobs, _ := s.repo.CountJobsByHrID(params.HrID)
	stats["total_jobs"] = totalJobs
	onlineJobs, _ := s.repo.CountOnlineJobsByHrID(params.HrID)
	stats["online_jobs"] = onlineJobs
	totalApps, _ := s.repo.CountApplicationsByHrID(params.HrID)
	stats["total_applications"] = totalApps

	for k, v := range stats {
		val, _ := json.Marshal(v)
		s.repo.SaveDashboardStat(params.HrID, k, string(val))
	}

	s.repo.CacheDelByPrefix(ctx, fmt.Sprintf("stats:hot_jobs:%d:", params.HrID))
	s.repo.CacheDelByPrefix(ctx, "stats:hot_skills:")

	resultJSON, _ := json.Marshal(stats)
	task.Result = string(resultJSON)
	return nil
}

func (s *TaskService) processNotification(ctx context.Context, task *model.AsyncTask) error {
	var params struct {
		UserID   uint64 `json:"user_id"`
		UserRole string `json:"user_role"`
		Title    string `json:"title"`
		Content  string `json:"content"`
		Channel  string `json:"channel"`
	}
	if err := json.Unmarshal([]byte(task.Params), &params); err != nil {
		return err
	}

	log.Printf("Notification sent to %s(%d): [%s] %s", params.UserRole, params.UserID, params.Title, params.Content)
	task.Result = fmt.Sprintf(`{"user_id":%d,"channel":"%s","sent":true}`, params.UserID, params.Channel)
	return nil
}
