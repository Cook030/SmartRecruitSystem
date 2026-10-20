package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"logic-grpc-service/recruitment"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

const (
	maxToolCalls       = 8
	maxToolResultBytes = 32 * 1024
	maxTotalToolBytes  = 96 * 1024
)

// Tools are request-scoped: the model never receives an identity parameter and
// there is no shared mutable HR identity across concurrent conversations.
func recruitmentTools(reader recruitment.Reader, hrID uint64) ([]tool.BaseTool, error) {
	jobs, e1 := queryTool("search_jobs", "查询当前HR自己的岗位；按岗位标题、地点和状态筛选，包含岗位要求。需要岗位ID时先用此工具定位。", func(ctx context.Context, q recruitment.JobQuery) (*recruitment.List[recruitment.Job], error) {
		return reader.SearchJobs(ctx, hrID, q)
	})
	candidates, e2 := queryTool("search_candidates", "筛选投递过当前HR岗位的候选人，按候选人去重。技能为全部包含的文字匹配；日期指投递时间。total是完整条件下总数，items仅当前页。", func(ctx context.Context, q recruitment.CandidateQuery) (*recruitment.List[recruitment.Candidate], error) {
		return reader.SearchCandidates(ctx, hrID, q)
	})
	detail, e3 := queryTool("get_candidate_detail", "获取当前HR可访问的单个候选人档案，包括工作/项目经历文字。不包含简历文件正文。", func(ctx context.Context, q recruitment.CandidateDetailQuery) (*recruitment.CandidateDetail, error) {
		return reader.GetCandidateDetail(ctx, hrID, q)
	})
	applications, e4 := queryTool("list_applications", "查询当前HR岗位的投递明细，含投递ID、候选人ID、岗位ID、状态和投递时间（上海时区）。", func(ctx context.Context, q recruitment.ApplicationQuery) (*recruitment.List[recruitment.Application], error) {
		return reader.ListRecruitmentApplications(ctx, hrID, q)
	})
	stats, e5 := queryTool("get_recruitment_stats", "统计当前HR岗位的投递数和去重候选人数。支持按岗位热度或学历分组；日期指投递时间；岗位热度包括零投递岗位。统计人数必须用此工具或列表total，不可数分页items推断总数。", func(ctx context.Context, q recruitment.StatsQuery) (*recruitment.Stats, error) {
		return reader.GetRecruitmentStats(ctx, hrID, q)
	})
	if err := errors.Join(e1, e2, e3, e4, e5); err != nil {
		return nil, err
	}
	return []tool.BaseTool{jobs, candidates, detail, applications, stats}, nil
}

func queryTool[Q, R any](name, description string, fn func(context.Context, Q) (R, error)) (tool.InvokableTool, error) {
	return utils.InferTool(name, description, fn, utils.WithUnmarshalArguments(func(_ context.Context, raw string) (any, error) {
		if len(raw) > 4096 || !strings.HasPrefix(strings.TrimSpace(raw), "{") {
			return nil, &recruitment.ValidationError{Message: "arguments must be a JSON object of at most 4096 bytes"}
		}
		var q Q
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&q); err != nil {
			return nil, &recruitment.ValidationError{Message: "invalid arguments: " + err.Error()}
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			return nil, &recruitment.ValidationError{Message: "arguments must contain one JSON object"}
		}
		validator, ok := any(&q).(interface{ Validate() error })
		if !ok {
			return nil, fmt.Errorf("query type %T does not implement Validate", q)
		}
		if err := validator.Validate(); err != nil {
			return nil, err
		}
		return q, nil
	}))
}

func toolError(code, message string) *compose.ToolOutput {
	b, _ := json.Marshal(map[string]any{"ok": false, "error": map[string]string{"code": code, "message": message}})
	return &compose.ToolOutput{Result: string(b)}
}

type queryBudget struct {
	calls atomic.Int32
	bytes atomic.Int64
}

// SQL errors never enter the model context. Cancellation and budget failures
// abort execution; recoverable errors are structured tool results, not empty data.
func (b *queryBudget) middleware(requestID string, hrID uint64) compose.InvokableToolMiddleware {
	return func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
		return func(ctx context.Context, input *compose.ToolInput) (output *compose.ToolOutput, err error) {
			started := time.Now()
			code := "ok"
			defer func() {
				size := 0
				rows := 0
				if output != nil {
					size = len(output.Result)
					var data struct {
						Items  []json.RawMessage `json:"items"`
						Groups []json.RawMessage `json:"groups"`
					}
					if code == "ok" && json.Unmarshal([]byte(output.Result), &data) == nil {
						rows = 1
						if data.Items != nil {
							rows = len(data.Items)
						} else if data.Groups != nil {
							rows = len(data.Groups)
						}
					}
				}
				slog.InfoContext(ctx, "recruitment tool", "request_id", requestID, "hr_id", hrID, "tool", input.Name,
					"call_id", input.CallID, "argument_bytes", len(input.Arguments), "result_bytes", size, "row_count", rows, "duration_ms", time.Since(started).Milliseconds(), "code", code)
			}()
			if err := ctx.Err(); err != nil {
				code = "cancelled"
				return nil, err
			}
			if b.calls.Add(1) > maxToolCalls {
				code = "budget_exceeded"
				return nil, errors.New("tool call budget exceeded")
			}
			output, err = next(ctx, input)
			if err != nil {
				if ctx.Err() != nil {
					code = "cancelled"
					return nil, ctx.Err()
				}
				var validation *recruitment.ValidationError
				switch {
				case errors.As(err, &validation):
					code = "invalid_arguments"
					return toolError(code, validation.Error()), nil
				case errors.Is(err, recruitment.ErrNotFound):
					code = "not_found"
					return toolError(code, "记录不存在或不在当前HR的访问范围内"), nil
				default:
					code = "query_failed"
					return toolError(code, "查询失败，无法据此判断是否有数据；请稍后重试"), nil
				}
			}
			if output == nil {
				code = "query_failed"
				return toolError(code, "查询未返回有效结果"), nil
			}
			if len(output.Result) > maxToolResultBytes {
				code = "result_too_large"
				return toolError(code, "结果超过大小限制，请缩小page_size或查询范围；没有返回该批数据"), nil
			}
			if b.bytes.Add(int64(len(output.Result))) > maxTotalToolBytes {
				code = "budget_exceeded"
				return nil, errors.New("tool result budget exceeded")
			}
			return output, nil
		}
	}
}
