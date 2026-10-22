package service

import (
	"context"

	"logic-grpc-service/model"
	"logic-grpc-service/pb"
	"logic-grpc-service/repository"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	repo *repository.Repository
}

func NewAuthService(repo *repository.Repository) *AuthService {
	return &AuthService{repo: repo}
}

func (s *AuthService) HrRegister(ctx context.Context, req *pb.HrRegisterRequest) (*pb.HrRegisterResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}

	if _, err := s.repo.GetCandidateByUsername(req.Username); err == nil {
		return nil, status.Error(codes.AlreadyExists, "该用户名已被候选人注册，无法再注册为HR")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	hr := &model.HrUser{
		Username: req.Username,
		Password: string(hashed),
		Name:     req.Name,
	}
	if err := s.repo.CreateHrUser(hr); err != nil {
		return nil, status.Error(codes.AlreadyExists, "username already exists")
	}

	return &pb.HrRegisterResponse{Id: hr.ID}, nil
}

func (s *AuthService) HrLogin(ctx context.Context, req *pb.HrLoginRequest) (*pb.HrLoginResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}

	hr, err := s.repo.GetHrUserByUsername(req.Username)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hr.Password), []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "wrong password")
	}

	return &pb.HrLoginResponse{
		Id:       hr.ID,
		Username: hr.Username,
		Name:     hr.Name,
	}, nil
}

func (s *AuthService) CandidateRegister(ctx context.Context, req *pb.CandidateRegisterRequest) (*pb.CandidateRegisterResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}

	if _, err := s.repo.GetHrUserByUsername(req.Username); err == nil {
		return nil, status.Error(codes.AlreadyExists, "该用户名已被HR注册，无法再注册为候选人")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	candidate := &model.Candidate{
		Username: req.Username,
		Password: string(hashed),
	}
	if err := s.repo.CreateCandidate(candidate); err != nil {
		return nil, status.Error(codes.AlreadyExists, "username already exists")
	}

	profile := &model.CandidateProfile{CandidateID: candidate.ID}
	_ = s.repo.UpsertProfile(profile)

	return &pb.CandidateRegisterResponse{Id: candidate.ID}, nil
}

func (s *AuthService) CandidateLogin(ctx context.Context, req *pb.CandidateLoginRequest) (*pb.CandidateLoginResponse, error) {
	scoped := *s
	scoped.repo = s.repo.WithContext(ctx)
	s = &scoped
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}

	candidate, err := s.repo.GetCandidateByUsername(req.Username)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(candidate.Password), []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "wrong password")
	}

	return &pb.CandidateLoginResponse{
		Id:       candidate.ID,
		Username: candidate.Username,
	}, nil
}
