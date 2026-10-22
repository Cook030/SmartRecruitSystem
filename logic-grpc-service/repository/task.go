package repository

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"logic-grpc-service/model"
	"logic-grpc-service/telemetry"
)

func (r *Repository) CreateAsyncTask(task *model.AsyncTask) error {
	return r.DB.Create(task).Error
}

func (r *Repository) CreateAndQueueTask(ctx context.Context, task *model.AsyncTask) (retErr error) {
	ctx, end := telemetry.Start(ctx, "task.create", attribute.String("task_type", telemetry.TaskType(task.TaskType)))
	defer func() { end(telemetry.Outcome(retErr)) }()
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	task.TraceParent, task.TraceState = carrier.Get("traceparent"), carrier.Get("tracestate")
	if err := r.DB.WithContext(ctx).Create(task).Error; err != nil {
		return err
	}
	if r.Redis.Available() {
		if err := r.Redis.LPush(ctx, "task:queue", fmt.Sprintf("%d", task.ID)); err != nil {
			telemetry.Event(ctx, "task.enqueue_failed")
			fmt.Printf("failed to push task %d to redis queue: %v\n", task.ID, err)
		}
	}
	return nil
}

func (r *Repository) GetAsyncTaskByID(id uint64) (*model.AsyncTask, error) {
	var task model.AsyncTask
	if err := r.DB.First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *Repository) UpdateAsyncTask(ctx context.Context, task *model.AsyncTask) error {
	err := r.DB.WithContext(ctx).Save(task).Error
	if err != nil {
		telemetry.Event(ctx, "task.state_update_failed")
	}
	return err
}

func (r *Repository) ListPendingTasks(limit int) ([]model.AsyncTask, error) {
	var tasks []model.AsyncTask
	err := r.DB.Where("status IN ?", []string{"pending", "retrying"}).
		Order("created_at ASC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

func (r *Repository) SaveResumeParseResult(result *model.ResumeParseResult) error {
	var existing model.ResumeParseResult
	err := r.DB.Where("resume_id = ?", result.ResumeID).First(&existing).Error
	if err == nil {
		// 记录已存在，执行更新
		existing.CandidateID = result.CandidateID
		existing.Education = result.Education
		existing.Experience = result.Experience
		existing.Projects = result.Projects
		existing.Skills = result.Skills
		existing.RawResult = result.RawResult
		return r.DB.Save(&existing).Error
	}
	// 记录不存在，执行插入
	return r.DB.Create(result).Error
}

func (r *Repository) GetResumeParseResultByResumeID(resumeID uint64) (*model.ResumeParseResult, error) {
	var result model.ResumeParseResult
	if err := r.DB.Where("resume_id = ?", resumeID).First(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *Repository) DeleteResumeParseResultByResumeID(resumeID uint64) error {
	return r.DB.Where("resume_id = ?", resumeID).Delete(&model.ResumeParseResult{}).Error
}

func (r *Repository) SaveAIMatchResult(result *model.AIMatchResult) error {
	return r.DB.Save(result).Error
}

func (r *Repository) GetAIMatchResultByApplicationID(appID uint64) (*model.AIMatchResult, error) {
	var result model.AIMatchResult
	if err := r.DB.Where("application_id = ?", appID).First(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *Repository) CountAIMatchResultsByHrID(hrID uint64) (int64, int64, int64, float64, error) {
	var total, success, failed int64
	var avgScore float64

	r.DB.Table("ai_match_results").
		Joins("JOIN jobs ON jobs.id = ai_match_results.job_id").
		Where("jobs.hr_id = ?", hrID).
		Count(&total)

	r.DB.Table("ai_match_results").
		Joins("JOIN jobs ON jobs.id = ai_match_results.job_id").
		Where("jobs.hr_id = ? AND ai_match_results.match_score > 0", hrID).
		Count(&success)

	r.DB.Table("ai_match_results").
		Joins("JOIN jobs ON jobs.id = ai_match_results.job_id").
		Where("jobs.hr_id = ? AND ai_match_results.match_score = 0", hrID).
		Count(&failed)

	r.DB.Table("ai_match_results").
		Joins("JOIN jobs ON jobs.id = ai_match_results.job_id").
		Where("jobs.hr_id = ?", hrID).
		Select("COALESCE(AVG(match_score), 0)").
		Scan(&avgScore)

	return total, success, failed, avgScore, nil
}

func (r *Repository) SaveDashboardStat(hrID uint64, statKey, statValue string) error {
	var stat model.DashboardStat
	err := r.DB.Where("hr_id = ? AND stat_key = ?", hrID, statKey).First(&stat).Error
	if err != nil {
		stat = model.DashboardStat{HrID: hrID, StatKey: statKey, StatValue: statValue}
		return r.DB.Create(&stat).Error
	}
	stat.StatValue = statValue
	stat.UpdatedAt = time.Now()
	return r.DB.Save(&stat).Error
}

func (r *Repository) GetDashboardStat(hrID uint64, statKey string) (*model.DashboardStat, error) {
	var stat model.DashboardStat
	if err := r.DB.Where("hr_id = ? AND stat_key = ?", hrID, statKey).First(&stat).Error; err != nil {
		return nil, err
	}
	return &stat, nil
}
