package service

import (
	"context"
	"fmt"
	"time"

	"logic-grpc-service/model"
	"logic-grpc-service/pb"
	"logic-grpc-service/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type JobService struct {
	pb.UnimplementedJobServiceServer
	repo *repository.Repository
}

func NewJobService(repo *repository.Repository) *JobService {
	return &JobService{repo: repo}
}

func (s *JobService) invalidateJobCache(ctx context.Context, hrID uint64) {
	s.repo.CacheDelByPrefix(ctx, "jobs:public:list:")
	s.repo.CacheDelByPrefix(ctx, fmt.Sprintf("stats:hot_jobs:%d:", hrID))
	s.repo.CacheDelByPrefix(ctx, "stats:hot_skills:")
}

func (s *JobService) CreateJob(ctx context.Context, req *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}

	job := &model.Job{
		HrID:        req.HrId,
		Title:       req.Title,
		Description: req.Description,
		Requirement: req.Requirement,
		Salary:      req.Salary,
		Location:    req.Location,
		Status:      1,
	}
	if err := s.repo.CreateJob(job); err != nil {
		return nil, status.Error(codes.Internal, "failed to create job")
	}

	s.invalidateJobCache(ctx, req.HrId)
	return &pb.CreateJobResponse{Id: job.ID}, nil
}

func (s *JobService) UpdateJob(ctx context.Context, req *pb.UpdateJobRequest) (*pb.UpdateJobResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	job, err := s.repo.GetJobByID(req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "job not found")
	}
	if job.HrID != req.HrId {
		return nil, status.Error(codes.PermissionDenied, "no permission to update this job")
	}

	job.Title = req.Title
	job.Description = req.Description
	job.Requirement = req.Requirement
	job.Salary = req.Salary
	job.Location = req.Location

	if err := s.repo.UpdateJob(job); err != nil {
		return nil, status.Error(codes.Internal, "failed to update job")
	}

	s.repo.CacheDel(ctx, fmt.Sprintf("job:detail:%d", req.Id))
	s.invalidateJobCache(ctx, req.HrId)
	return &pb.UpdateJobResponse{}, nil
}

func (s *JobService) OfflineJob(ctx context.Context, req *pb.OfflineJobRequest) (*pb.OfflineJobResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	job, err := s.repo.GetJobByID(req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "job not found")
	}
	if job.HrID != req.HrId {
		return nil, status.Error(codes.PermissionDenied, "no permission to offline this job")
	}

	if err := s.repo.OfflineJob(req.Id, req.HrId); err != nil {
		return nil, status.Error(codes.Internal, "failed to offline job")
	}

	s.repo.CacheDel(ctx, fmt.Sprintf("job:detail:%d", req.Id))
	s.invalidateJobCache(ctx, req.HrId)
	return &pb.OfflineJobResponse{}, nil
}

func (s *JobService) ListHrJobs(ctx context.Context, req *pb.ListHrJobsRequest) (*pb.ListHrJobsResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	page, pageSize := normalizePageParams(int(req.Page), int(req.PageSize))
	jobs, total, err := s.repo.ListJobsByHrID(req.HrId, page, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list jobs")
	}
	return &pb.ListHrJobsResponse{
		Jobs:  toJobInfoList(jobs),
		Total: total,
	}, nil
}

type cachedJobList struct {
	Jobs  []model.Job `json:"jobs"`
	Total int64       `json:"total"`
}

func (s *JobService) ListPublicJobs(ctx context.Context, req *pb.ListPublicJobsRequest) (*pb.ListPublicJobsResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	page, pageSize := normalizePageParams(int(req.Page), int(req.PageSize))
	key := fmt.Sprintf("jobs:public:list:page:%d:size:%d", page, pageSize)
	var cached cachedJobList
	err := s.repo.CacheGet(ctx, key, &cached, 5*time.Minute, func() (interface{}, error) {
		jobs, total, err := s.repo.ListPublicJobs(page, pageSize)
		if err != nil {
			return nil, err
		}
		return cachedJobList{Jobs: jobs, Total: total}, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list jobs")
	}
	return &pb.ListPublicJobsResponse{
		Jobs:  toJobInfoList(cached.Jobs),
		Total: cached.Total,
	}, nil
}

func (s *JobService) GetJob(ctx context.Context, req *pb.GetJobRequest) (*pb.GetJobResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	key := fmt.Sprintf("job:detail:%d", req.Id)
	var job model.Job
	err := s.repo.CacheGet(ctx, key, &job, 5*time.Minute, func() (interface{}, error) {
		j, err := s.repo.GetJobByID(req.Id)
		if err != nil {
			return nil, err
		}
		return *j, nil
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, "job not found")
	}
	return &pb.GetJobResponse{Job: toJobInfo(&job)}, nil
}

func toJobInfo(j *model.Job) *pb.JobInfo {
	return &pb.JobInfo{
		Id:          j.ID,
		HrId:        j.HrID,
		Title:       j.Title,
		Description: j.Description,
		Requirement: j.Requirement,
		Salary:      j.Salary,
		Location:    j.Location,
		Status:      int32(j.Status),
		CreatedAt:   j.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   j.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func toJobInfoList(jobs []model.Job) []*pb.JobInfo {
	list := make([]*pb.JobInfo, len(jobs))
	for i := range jobs {
		list[i] = toJobInfo(&jobs[i])
	}
	return list
}

func normalizePageParams(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return page, pageSize
}
