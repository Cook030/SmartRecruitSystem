package chatcontext

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func handoff() *Handoff {
	return &Handoff{Goal: "岗位123筛选Go候选人", Completed: "已定位岗位，人数为历史快照", Pending: "比较候选人", Decisions: "不限学历，已取消本科条件", Failures: "同名查询有歧义，需ID", Safety: "等待候选人ID，仅查询"}
}

func history(n int) []Message {
	var messages []Message
	for i := 0; i < n; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		messages = append(messages, Message{ID: uint64(i + 1), Role: role, Content: "正文"})
	}
	return messages
}

func loader(all []Message) func(context.Context, uint64, uint64) ([]Message, error) {
	return func(ctx context.Context, after, before uint64) ([]Message, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var page []Message
		for _, m := range all {
			if m.ID > after && m.ID < before {
				page = append(page, m)
			}
			if len(page) == BatchLimit {
				break
			}
		}
		return page, nil
	}
}

func TestHandoffValidation(t *testing.T) {
	h := handoff()
	got, err := ParseHandoff(h.JSON())
	if err != nil || !reflect.DeepEqual(h, got) {
		t.Fatalf("roundtrip: %v, %v", got, err)
	}
	for _, raw := range []string{
		`null`, `{}`, `{"current_goal":"x"}`, h.JSON() + ` {}`,
		strings.Replace(h.JSON(), `"safe_stopping_point"`, `"unexpected"`, 1),
		strings.Replace(h.JSON(), `"pending_tasks":"比较候选人"`, `"pending_tasks":" "`, 1),
		strings.Repeat("x", MaxHandoffBytes+1),
	} {
		if _, err := ParseHandoff(raw); err == nil {
			t.Fatalf("accepted invalid handoff %q", raw[:min(len(raw), 80)])
		}
	}
}

func TestSelectPreservesWholeTurns(t *testing.T) {
	all := history(6)
	pairBytes := len(all[4].Content) + len(all[5].Content) + 64
	got := Select(all, pairBytes)
	if !reflect.DeepEqual(got, all[4:]) {
		t.Fatalf("wrong suffix: %v", got)
	}
	if len(Select(all, pairBytes-1)) != 0 {
		t.Fatal("split an oversized turn")
	}
	all[5].Role = "tool"
	if len(Select(all, 10000)) != 0 {
		t.Fatal("accepted incomplete tool turn")
	}
}

func TestPrepareBackfillsBeyondRecentWindow(t *testing.T) {
	all := history(100)
	var seen []Message
	out, err := Prepare(context.Background(), Input{Recent: all[60:], Load: loader(all)}, 10000,
		func(_ context.Context, old *Handoff, batch []Message) (*Handoff, error) {
			if len(seen) > 0 && old == nil {
				t.Fatal("lost previous handoff")
			}
			seen = append(seen, batch...)
			return handoff(), nil
		})
	if err != nil || out.Incomplete || out.Update == nil {
		t.Fatalf("prepare: %+v, %v", out, err)
	}
	if !reflect.DeepEqual(seen, all[:60]) || out.Update.UntilID != 60 || out.Update.Version != 1 {
		t.Fatal("skipped or duplicated old history")
	}
	if !reflect.DeepEqual(out.Recent, all[60:]) {
		t.Fatal("changed recent turns")
	}
	messages := out.Structure("继续比较")
	if messages[0].Role != "user" || !strings.Contains(messages[0].Content, handoff().Safety) || messages[len(messages)-1].Content != "继续比较" {
		t.Fatal("incorrect handoff authority or question order")
	}

	// Persisted cursor resumes exactly once without summarizing retained turns.
	next, err := Prepare(context.Background(), Input{State: *out.Update, Recent: all[60:], Load: loader(all)}, 10000,
		func(context.Context, *Handoff, []Message) (*Handoff, error) {
			t.Fatal("repeated compaction")
			return nil, nil
		})
	if err != nil || next.Update != nil || next.Incomplete {
		t.Fatalf("resume: %+v %v", next, err)
	}
}

func TestPrepareFailureKeepsLastSuccessfulCursor(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		all := history(80)
		calls := 0
		out, err := Prepare(context.Background(), Input{State: State{Handoff: handoff(), Version: 7}, Recent: all[40:], Load: loader(all)}, 10000,
			func(context.Context, *Handoff, []Message) (*Handoff, error) {
				calls++
				if calls == 1 {
					return handoff(), nil
				}
				if invalid {
					return &Handoff{Goal: "incomplete"}, nil
				}
				return nil, errors.New("provider unavailable")
			})
		if err == nil || !out.Incomplete || out.Update == nil || out.Update.UntilID != 20 || out.Update.Version != 8 {
			t.Fatalf("advanced across failure: %+v %v", out, err)
		}
	}
}

func TestPrepareBoundedBacklogAndFailureFallback(t *testing.T) {
	all := history(160)
	out, err := Prepare(context.Background(), Input{Recent: all[120:], Load: loader(all)}, 10000,
		func(context.Context, *Handoff, []Message) (*Handoff, error) { return handoff(), nil })
	if err != nil || !out.Incomplete || out.Update.UntilID != 80 {
		t.Fatalf("unbounded backlog: %+v %v", out, err)
	}

	for _, mode := range []string{"cancelled", "oversized", "malformed", "provider"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			all := history(4)
			if mode == "cancelled" {
				cancel()
			}
			if mode == "oversized" {
				all[0].Content = strings.Repeat("x", maxBatchBytes)
			}
			if mode == "malformed" {
				all[1].Role = "user"
			}
			old := handoff()
			out, err := Prepare(ctx, Input{State: State{Handoff: old, Version: 3}, Recent: all, Load: loader(all)}, 0,
				func(context.Context, *Handoff, []Message) (*Handoff, error) {
					return nil, errors.New("provider unavailable")
				})
			if err == nil || !out.Incomplete || out.Update != nil || out.Handoff != old {
				t.Fatalf("unsafe fallback: %+v %v", out, err)
			}
		})
	}
}
