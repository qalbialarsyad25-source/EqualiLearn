package service

import (
	"context"
	"testing"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"
	"EquiliLearn/pkg/gemini"

	"github.com/google/uuid"
)

type mockDocumentSummaryRepo struct {
	summaries []entity.DocumentSummary
}

func (m *mockDocumentSummaryRepo) CreateDocumentSummary(ctx context.Context, summary *entity.DocumentSummary) error {
	m.summaries = append(m.summaries, *summary)
	return nil
}

func (m *mockDocumentSummaryRepo) UpdateDocumentSummary(ctx context.Context, summary *entity.DocumentSummary) error {
	for i, s := range m.summaries {
		if s.ID == summary.ID {
			m.summaries[i] = *summary
			return nil
		}
	}
	m.summaries = append(m.summaries, *summary)
	return nil
}

func (m *mockDocumentSummaryRepo) GetSummariesByUserID(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]entity.DocumentSummary, int64, error) {
	var results []entity.DocumentSummary
	for _, s := range m.summaries {
		if s.UserID != nil && *s.UserID == userID {
			results = append(results, s)
		}
	}
	return results, int64(len(results)), nil
}

func (m *mockDocumentSummaryRepo) GetSummaryByID(ctx context.Context, id uuid.UUID) (*entity.DocumentSummary, error) {
	for _, s := range m.summaries {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, nil
}

func (m *mockDocumentSummaryRepo) DeleteSummary(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	var filtered []entity.DocumentSummary
	for _, s := range m.summaries {
		if !(s.ID == id && s.UserID != nil && *s.UserID == userID) {
			filtered = append(filtered, s)
		}
	}
	m.summaries = filtered
	return nil
}

func TestDocumentSummaryService_SummarizeDocument(t *testing.T) {
	mockClient := gemini.NewMockGeminiClient()
	repo := &mockDocumentSummaryRepo{}
	svc := NewDocumentSummaryService(mockClient, repo)

	ctx := context.Background()
	userID := uuid.New()

	req := model.SummarizeDocumentRequest{
		Language:       "id",
		DetailLevel:    "balanced",
		TargetAudience: "student",
	}

	rawDoc := []byte("Pembelajaran Mesin adalah bidang dalam kecerdasan buatan yang memungkinkan sistem untuk belajar secara mandiri dari data.")
	res, err := svc.SummarizeDocument(ctx, req, rawDoc, "ai_lecture.txt", "text/plain", &userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res == nil {
		t.Fatal("expected non-nil response")
	}

	if res.Title == "" {
		t.Error("expected title in response")
	}

	if res.Summary == "" {
		t.Error("expected summary in response")
	}

	if len(res.KeyPoints) == 0 {
		t.Error("expected key points in response")
	}

	if res.Explanation == "" {
		t.Error("expected explanation in response")
	}

	// Verify it was stored in repo
	if len(repo.summaries) != 1 {
		t.Fatalf("expected 1 summary in repo, got %d", len(repo.summaries))
	}
}

func TestDocumentSummaryService_SummarizeText(t *testing.T) {
	mockClient := gemini.NewMockGeminiClient()
	repo := &mockDocumentSummaryRepo{}
	svc := NewDocumentSummaryService(mockClient, repo)

	ctx := context.Background()
	userID := uuid.New()

	req := model.SummarizeTextRequest{
		Title:          "Calculus Fundamentals",
		Text:           "Differentiation is a method to compute the rate at which a dependent output y changes with respect to the change in the independent input x.",
		Language:       "en",
		DetailLevel:    "brief",
		TargetAudience: "student",
	}

	res, err := svc.SummarizeText(ctx, req, &userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.Title != "Calculus Fundamentals" {
		t.Errorf("expected title 'Calculus Fundamentals', got %q", res.Title)
	}

	if res.Summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestDocumentSummaryService_UpdateSummary(t *testing.T) {
	mockClient := gemini.NewMockGeminiClient()
	repo := &mockDocumentSummaryRepo{}
	svc := NewDocumentSummaryService(mockClient, repo)

	ctx := context.Background()
	userID := uuid.New()

	created, err := svc.SummarizeText(ctx, model.SummarizeTextRequest{
		Title: "Initial Title",
		Text:  "Original content text to summarize.",
	}, &userID)
	if err != nil {
		t.Fatalf("failed to create initial summary: %v", err)
	}

	newTitle := "Updated Advanced Physics Title"
	newSummary := "Edited summary text with customized key insights."
	newKeyPoints := []string{"Key point 1 updated", "Key point 2 updated"}
	newExplanation := "Custom user explanation notes."

	updateReq := model.UpdateDocumentSummaryRequest{
		Title:       &newTitle,
		Summary:     &newSummary,
		KeyPoints:   &newKeyPoints,
		Explanation: &newExplanation,
	}

	updated, err := svc.UpdateSummary(ctx, created.ID, userID, updateReq)
	if err != nil {
		t.Fatalf("failed to update summary: %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, updated.Title)
	}
	if updated.Summary != newSummary {
		t.Errorf("expected summary %q, got %q", newSummary, updated.Summary)
	}
	if len(updated.KeyPoints) != 2 || updated.KeyPoints[0] != "Key point 1 updated" {
		t.Errorf("expected updated key points, got %+v", updated.KeyPoints)
	}
	if updated.Explanation != newExplanation {
		t.Errorf("expected explanation %q, got %q", newExplanation, updated.Explanation)
	}

	// Test unauthorized update
	otherUserID := uuid.New()
	_, err = svc.UpdateSummary(ctx, created.ID, otherUserID, updateReq)
	if err == nil {
		t.Fatal("expected error on unauthorized user update")
	}
}

func TestDocumentSummaryService_GetHistoryAndDelete(t *testing.T) {
	mockClient := gemini.NewMockGeminiClient()
	repo := &mockDocumentSummaryRepo{}
	svc := NewDocumentSummaryService(mockClient, repo)

	ctx := context.Background()
	userID := uuid.New()

	// Create 2 summaries
	_, _ = svc.SummarizeText(ctx, model.SummarizeTextRequest{Title: "Note 1", Text: "Content 1"}, &userID)
	res2, _ := svc.SummarizeText(ctx, model.SummarizeTextRequest{Title: "Note 2", Text: "Content 2"}, &userID)

	items, total, err := svc.GetSummariesByUserID(ctx, userID, model.Pagination{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if total != 2 || len(items) != 2 {
		t.Fatalf("expected 2 items, got %d (total: %d)", len(items), total)
	}

	// Delete 1 summary
	err = svc.DeleteSummary(ctx, res2.ID, userID)
	if err != nil {
		t.Fatalf("failed to delete summary: %v", err)
	}

	itemsAfter, totalAfter, _ := svc.GetSummariesByUserID(ctx, userID, model.Pagination{Page: 1, Limit: 10})
	if totalAfter != 1 || len(itemsAfter) != 1 {
		t.Fatalf("expected 1 item after deletion, got %d (total: %d)", len(itemsAfter), totalAfter)
	}
}
