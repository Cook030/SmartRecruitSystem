package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"logic-grpc-service/chatcontext"
	"logic-grpc-service/config"
	"logic-grpc-service/recruitment"
)

// Embedded Reader makes accidental other queries fail instead of inventing data.
type failingStats struct {
	recruitment.Reader
	calls atomic.Int32
}

func (r *failingStats) GetRecruitmentStats(_ context.Context, hrID uint64, _ recruitment.StatsQuery) (*recruitment.Stats, error) {
	if hrID != 42 {
		return nil, errors.New("wrong HR identity")
	}
	r.calls.Add(1)
	return nil, errors.New("private SQL failure must not enter history")
}

func TestChatCompactionAndToolFailure(t *testing.T) {
	for _, valid := range []bool{true, false} {
		t.Run(map[bool]string{true: "handoff", false: "fallback"}[valid], func(t *testing.T) {
			h := &chatcontext.Handoff{Goal: "岗位123", Completed: "已定位岗位，历史快照", Pending: "统计", Decisions: "不限学历", Failures: "无", Safety: "仅查询"}
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					Messages []struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"messages"`
					Tools []json.RawMessage `json:"tools"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				n := requests.Add(1)
				message := map[string]any{"role": "assistant"}
				finish := "stop"
				switch n {
				case 1:
					if len(request.Tools) != 0 || request.Messages[0].Content != handoffPrompt {
						t.Error("summarizer received recruitment tools or wrong system rules")
					}
					message["content"] = h.JSON()
					if !valid {
						message["content"] = `{"current_goal":"missing five fields"}`
					}
				case 2:
					if len(request.Tools) != 5 {
						t.Error("agent tools missing")
					}
					if valid {
						if !strings.Contains(request.Messages[1].Content, h.JSON()) || request.Messages[1].Role != "user" {
							t.Error("handoff missing or elevated to system authority")
						}
					} else if !strings.Contains(request.Messages[0].Content, "尚未成功压缩") {
						t.Error("fallback warning missing")
					}
					message["tool_calls"] = []any{map[string]any{"id": "call_stats", "type": "function", "function": map[string]any{"name": "get_recruitment_stats", "arguments": `{"job_id":123}`}}}
					finish = "tool_calls"
				case 3:
					last := request.Messages[len(request.Messages)-1]
					if last.Role != "tool" || !strings.Contains(last.Content, "query_failed") || strings.Contains(last.Content, "private SQL") {
						t.Error("tool error was not sanitized")
					}
					message["content"] = "查询失败，无法确认人数，请稍后重试。"
				default:
					t.Errorf("unexpected model call %d", n)
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"id": "test", "object": "chat.completion", "model": "test-model", "choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": finish}}})
			}))
			defer server.Close()
			all := []chatcontext.Message{{ID: 1, Role: "user", Content: strings.Repeat("早期条件", 2200)}, {ID: 2, Role: "assistant", Content: "已确认"}, {ID: 3, Role: "user", Content: "不限学历"}, {ID: 4, Role: "assistant", Content: "收到"}}
			input := chatcontext.Input{Recent: all, Load: func(_ context.Context, after, before uint64) ([]chatcontext.Message, error) {
				if after == 0 && before == 3 {
					return all[:2], nil
				}
				return nil, nil
			}}
			reader := &failingStats{}
			result, err := Chat(context.Background(), &config.Config{AI: config.AIConfig{BaseURL: server.URL + "/v1", Model: "test-model", APIKey: "test-key"}}, reader, 42, input, "查一下岗位123当前人数")
			if err != nil {
				t.Fatal(err)
			}
			if requests.Load() != 3 || reader.calls.Load() != 1 {
				t.Fatalf("calls: %d, queries: %d", requests.Load(), reader.calls.Load())
			}
			if valid && (result.ContextUpdate == nil || result.ContextUpdate.UntilID != 2) {
				t.Fatal("missing successful progress")
			}
			if !valid && result.ContextUpdate != nil {
				t.Fatal("invalid summary advanced progress")
			}
			if result.Reply == "" || !strings.Contains(result.ContextNotes, "get_recruitment_stats: query_failed") || strings.Contains(result.ContextNotes, "private SQL") {
				t.Fatalf("incorrect result: %+v", result)
			}
		})
	}
}
