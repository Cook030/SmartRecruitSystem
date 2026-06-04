package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"logic-grpc-service/config"
	"logic-grpc-service/model"
	"logic-grpc-service/pb"
	"logic-grpc-service/repository"

	ossClient "logic-grpc-service/oss"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CandidateService struct {
	pb.UnimplementedCandidateServiceServer
	repo *repository.Repository
	cfg  *config.Config
}

func NewCandidateService(repo *repository.Repository, cfg *config.Config) *CandidateService {
	return &CandidateService{repo: repo, cfg: cfg}
}

func (s *CandidateService) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	p, err := s.repo.GetProfile(req.CandidateId)
	if err != nil {
		return &pb.GetProfileResponse{Profile: &pb.ProfileInfo{CandidateId: req.CandidateId}}, nil
	}
	return &pb.GetProfileResponse{Profile: toProfileInfo(p)}, nil
}

func (s *CandidateService) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	isComplete := req.Name != "" && req.Phone != "" && req.Education != "" &&
		req.University != "" && req.Experience != "" && req.Skills != ""

	profile := &model.CandidateProfile{
		CandidateID: req.CandidateId,
		Name:        req.Name,
		Phone:       req.Phone,
		Education:   req.Education,
		University:  req.University,
		Experience:  req.Experience,
		Skills:      req.Skills,
		IsComplete:  isComplete,
	}
	if err := s.repo.UpsertProfile(profile); err != nil {
		return nil, status.Error(codes.Internal, "failed to update profile")
	}

	s.repo.CacheDelByPrefix(ctx, "dashboard:funnel:")
	s.repo.CacheDelByPrefix(ctx, "dashboard:hot_skills:")

	return &pb.UpdateProfileResponse{IsComplete: isComplete}, nil
}

func (s *CandidateService) UploadResume(ctx context.Context, req *pb.UploadResumeRequest) (*pb.UploadResumeResponse, error) {
	allowedTypes := map[string]bool{"pdf": true, "doc": true, "docx": true}
	fileType := strings.ToLower(req.FileType)
	if !allowedTypes[fileType] {
		return nil, status.Error(codes.InvalidArgument, "only pdf/doc/docx files are allowed")
	}

	if !validateFileHeader(req.FileData, fileType) {
		return nil, status.Error(codes.InvalidArgument, "file header does not match file type")
	}

	ossKey := fmt.Sprintf("resumes/%d/%d_%s", req.CandidateId, time.Now().UnixMilli(), req.FileName)

	if err := ossClient.UploadFile(s.cfg, ossKey, req.FileData); err != nil {
		return nil, status.Error(codes.Internal, "failed to upload to OSS: "+err.Error())
	}

	resume := &model.Resume{
		CandidateID: req.CandidateId,
		FileName:    req.FileName,
		OssKey:      ossKey,
		FileSize:    req.FileSize,
		FileType:    fileType,
	}
	if err := s.repo.CreateResume(resume); err != nil {
		return nil, status.Error(codes.Internal, "failed to save resume record")
	}

	task := &model.AsyncTask{
		TaskType:   "resume_parse",
		Status:     "pending",
		Params:     fmt.Sprintf(`{"resume_id":%d,"candidate_id":%d,"oss_key":"%s"}`, resume.ID, req.CandidateId, ossKey),
		MaxRetries: 3,
	}
	if err := s.repo.CreateAndQueueTask(task); err != nil {
		fmt.Printf("failed to create resume parse task: %v\n", err)
	}

	s.repo.CacheDelByPrefix(ctx, "dashboard:funnel:")

	return &pb.UploadResumeResponse{Id: resume.ID, OssKey: ossKey}, nil
}

func (s *CandidateService) GetResume(ctx context.Context, req *pb.GetResumeRequest) (*pb.GetResumeResponse, error) {
	resume, err := s.repo.GetResumeByCandidate(req.CandidateId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "resume not found")
	}

	signedURL, err := ossClient.GetSignedURL(s.cfg, resume.OssKey)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate signed URL")
	}

	return &pb.GetResumeResponse{
		Resume: &pb.ResumeInfo{
			Id:          resume.ID,
			CandidateId: resume.CandidateID,
			FileName:    resume.FileName,
			OssKey:      resume.OssKey,
			FileSize:    resume.FileSize,
			FileType:    resume.FileType,
			Url:         signedURL,
			CreatedAt:   resume.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}

func (s *CandidateService) DeleteResume(ctx context.Context, req *pb.DeleteResumeRequest) (*pb.DeleteResumeResponse, error) {
	resume, err := s.repo.GetResumeByCandidate(req.CandidateId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "resume not found")
	}

	if err := ossClient.DeleteFile(s.cfg, resume.OssKey); err != nil {
		fmt.Printf("failed to delete OSS file %s: %v\n", resume.OssKey, err)
	}

	if err := s.repo.DeleteResume(req.CandidateId); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete resume record")
	}

	s.repo.CacheDelByPrefix(ctx, "dashboard:funnel:")

	return &pb.DeleteResumeResponse{}, nil
}

func (s *CandidateService) ApplyJob(ctx context.Context, req *pb.ApplyJobRequest) (*pb.ApplyJobResponse, error) {
	profile, err := s.repo.GetProfile(req.CandidateId)
	if err != nil || !profile.IsComplete {
		return nil, status.Error(codes.FailedPrecondition, "please complete your profile first")
	}

	_, err = s.repo.GetResumeByCandidate(req.CandidateId)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "please upload your resume first")
	}

	job, err := s.repo.GetJobByID(req.JobId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "job not found")
	}
	if job.Status == 0 {
		return nil, status.Error(codes.FailedPrecondition, "job is offline")
	}

	existApp, _ := s.repo.GetApplicationByJobAndCandidate(req.JobId, req.CandidateId)
	if existApp != nil && existApp.ID > 0 {
		return nil, status.Error(codes.AlreadyExists, "you have already applied for this job")
	}

	app := &model.Application{
		JobID:       req.JobId,
		CandidateID: req.CandidateId,
		Status:      0,
	}
	if err := s.repo.CreateApplication(app); err != nil {
		return nil, status.Error(codes.Internal, "failed to create application")
	}

	matchTask := &model.AsyncTask{
		TaskType:   "ai_match",
		Status:     "pending",
		Params:     fmt.Sprintf(`{"application_id":%d,"job_id":%d,"candidate_id":%d}`, app.ID, req.JobId, req.CandidateId),
		MaxRetries: 3,
	}
	if err := s.repo.CreateAndQueueTask(matchTask); err != nil {
		fmt.Printf("failed to create ai match task: %v\n", err)
	}

	s.repo.CacheDelByPrefix(ctx, fmt.Sprintf("dashboard:overview:%d", job.HrID))
	s.repo.CacheDelByPrefix(ctx, fmt.Sprintf("dashboard:hot_jobs:%d", job.HrID))
	s.repo.CacheDelByPrefix(ctx, fmt.Sprintf("dashboard:funnel:%d", job.HrID))
	s.repo.CacheDelByPrefix(ctx, fmt.Sprintf("dashboard:app_trend:%d", job.HrID))

	return &pb.ApplyJobResponse{Id: app.ID}, nil
}

func (s *CandidateService) ListMyApplications(ctx context.Context, req *pb.ListMyApplicationsRequest) (*pb.ListMyApplicationsResponse, error) {
	page, pageSize := normalizePageParams(int(req.Page), int(req.PageSize))
	apps, total, err := s.repo.ListApplicationsByCandidate(req.CandidateId, page, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list applications")
	}

	var list []*pb.ApplicationInfo
	for _, app := range apps {
		info := &pb.ApplicationInfo{
			Id:          app.ID,
			JobId:       app.JobID,
			CandidateId: app.CandidateID,
			Status:      int32(app.Status),
			CreatedAt:   app.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		job, _ := s.repo.GetJobByID(app.JobID)
		if job != nil {
			info.JobTitle = job.Title
			info.JobLocation = job.Location
			info.JobSalary = job.Salary
		}
		list = append(list, info)
	}
	return &pb.ListMyApplicationsResponse{Applications: list, Total: total}, nil
}

func (s *CandidateService) ListJobApplications(ctx context.Context, req *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error) {
	job, err := s.repo.GetJobByID(req.JobId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "job not found")
	}
	if job.HrID != req.HrId {
		return nil, status.Error(codes.PermissionDenied, "no permission")
	}

	page, pageSize := normalizePageParams(int(req.Page), int(req.PageSize))
	apps, total, err := s.repo.ListApplicationsByJobID(req.JobId, page, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list applications")
	}

	var list []*pb.ApplicationInfo
	for _, app := range apps {
		info := &pb.ApplicationInfo{
			Id:          app.ID,
			JobId:       app.JobID,
			CandidateId: app.CandidateID,
			Status:      int32(app.Status),
			CreatedAt:   app.CreatedAt.Format("2006-01-02 15:04:05"),
			JobTitle:    job.Title,
		}
		profile, _ := s.repo.GetProfile(app.CandidateID)
		if profile != nil {
			info.CandidateName = profile.Name
		}
		list = append(list, info)
	}
	return &pb.ListJobApplicationsResponse{Applications: list, Total: total}, nil
}

func (s *CandidateService) GetApplicationDetail(ctx context.Context, req *pb.GetApplicationDetailRequest) (*pb.GetApplicationDetailResponse, error) {
	app, err := s.repo.GetApplicationByID(req.ApplicationId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "application not found")
	}

	job, err := s.repo.GetJobByID(app.JobID)
	if err != nil {
		return nil, status.Error(codes.Internal, "job not found")
	}
	if job.HrID != req.HrId {
		return nil, status.Error(codes.PermissionDenied, "no permission")
	}

	profile, _ := s.repo.GetProfile(app.CandidateID)

	var resumeInfo *pb.ResumeInfo
	resume, err := s.repo.GetResumeByCandidate(app.CandidateID)
	if err == nil {
		signedURL, _ := ossClient.GetSignedURL(s.cfg, resume.OssKey)
		resumeInfo = &pb.ResumeInfo{
			Id:          resume.ID,
			CandidateId: resume.CandidateID,
			FileName:    resume.FileName,
			OssKey:      resume.OssKey,
			FileSize:    resume.FileSize,
			FileType:    resume.FileType,
			Url:         signedURL,
			CreatedAt:   resume.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return &pb.GetApplicationDetailResponse{
		Application: &pb.ApplicationInfo{
			Id:          app.ID,
			JobId:       app.JobID,
			CandidateId: app.CandidateID,
			Status:      int32(app.Status),
			CreatedAt:   app.CreatedAt.Format("2006-01-02 15:04:05"),
			JobTitle:    job.Title,
		},
		Profile: toProfileInfo(profile),
		Resume:  resumeInfo,
	}, nil
}

func toProfileInfo(p *model.CandidateProfile) *pb.ProfileInfo {
	if p == nil {
		return &pb.ProfileInfo{}
	}
	return &pb.ProfileInfo{
		CandidateId: p.CandidateID,
		Name:        p.Name,
		Phone:       p.Phone,
		Education:   p.Education,
		University:  p.University,
		Experience:  p.Experience,
		Skills:      p.Skills,
		IsComplete:  p.IsComplete,
	}
}

func validateFileHeader(data []byte, fileType string) bool {
	if len(data) < 4 {
		return false
	}
	switch fileType {
	case "pdf":
		return bytes.HasPrefix(data, []byte("%PDF"))
	case "doc":
		return bytes.HasPrefix(data, []byte{0xD0, 0xCF, 0x11, 0xE0})
	case "docx":
		return bytes.HasPrefix(data, []byte{0x50, 0x4B, 0x03, 0x04})
	}
	return false
}
