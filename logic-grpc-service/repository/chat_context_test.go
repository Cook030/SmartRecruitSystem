package repository

import (
	"encoding/json"
	"strings"
	"testing"

	"logic-grpc-service/model"
)

func TestContextMessagesKeepsInternalFailuresAndLegacyRoles(t *testing.T) {
	stored := []model.ChatMessage{
		{ID: 1, Role: "hr", Content: "查询岗位123"},
		{ID: 2, Role: "ai", Content: "查询失败", ContextNotes: "search_jobs: query_failed"},
	}
	got := contextMessages(stored)
	if got[0].Role != "user" || got[1].Role != "assistant" || got[1].ID != 2 || !strings.Contains(got[1].Content, stored[1].ContextNotes) {
		t.Fatalf("lost legacy conversation or failure evidence: %+v", got)
	}
	visible, err := json.Marshal(stored[1])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(visible), "search_jobs") || stored[1].Content != "查询失败" {
		t.Fatal("internal notes leaked into visible reply")
	}
}
