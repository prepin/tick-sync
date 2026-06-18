package tickticktokenreminder

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/prepin/tick-sync/internal/infra/sqlite/oauthtokens"
	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
	"github.com/prepin/tick-sync/internal/usecase/tickticktokenreminder/mocks"
)

var reminderNow = time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

// Does not create a reminder when TickTick has not been connected yet.
func TestHandleIgnoresMissingToken(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	tokens := mocks.NewMockTokenRepository(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	tokens.EXPECT().
		Get(gomock.Any(), oauthtokens.ProviderTickTick).
		Return(oauthtokens.Token{}, oauthtokens.ErrTokenNotFound)

	uc := New(tokens, ticktick, WithNow(func() time.Time { return reminderNow }))
	if err := uc.Handle(t.Context()); err != nil {
		t.Fatalf("handle reminder: %v", err)
	}
}

// Does not create a reminder when the stored token has no known expiry.
func TestHandleIgnoresTokenWithoutExpiry(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	tokens := mocks.NewMockTokenRepository(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	tokens.EXPECT().Get(gomock.Any(), oauthtokens.ProviderTickTick).Return(tokenFixture(), nil)

	uc := New(tokens, ticktick, WithNow(func() time.Time { return reminderNow }))
	if err := uc.Handle(t.Context()); err != nil {
		t.Fatalf("handle reminder: %v", err)
	}
}

// Does not create a reminder when the stored token expires after the two-week reminder window.
func TestHandleIgnoresTokenExpiringAfterTwoWeeks(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	tokens := mocks.NewMockTokenRepository(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	token := tokenFixture()
	token.ExpiresAt = reminderNow.Add(14*24*time.Hour + time.Second)
	tokens.EXPECT().Get(gomock.Any(), oauthtokens.ProviderTickTick).Return(token, nil)

	uc := New(tokens, ticktick, WithNow(func() time.Time { return reminderNow }))
	if err := uc.Handle(t.Context()); err != nil {
		t.Fatalf("handle reminder: %v", err)
	}
}

// Does not create a duplicate reminder when the stored token already has a reminder task id.
func TestHandleIgnoresTokenWithExistingReminder(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	tokens := mocks.NewMockTokenRepository(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	token := tokenFixture()
	token.ExpiresAt = reminderNow.Add(24 * time.Hour)
	token.RefreshReminderTaskID = "task-1"
	tokens.EXPECT().Get(gomock.Any(), oauthtokens.ProviderTickTick).Return(token, nil)

	uc := New(tokens, ticktick, WithNow(func() time.Time { return reminderNow }))
	if err := uc.Handle(t.Context()); err != nil {
		t.Fatalf("handle reminder: %v", err)
	}
}

// Creates one medium-priority reminder without a due date when the stored token expires within two weeks.
func TestHandleCreatesReminderForTokenExpiringWithinTwoWeeks(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	tokens := mocks.NewMockTokenRepository(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	token := tokenFixture()
	token.ExpiresAt = reminderNow.Add(24 * time.Hour)
	tokens.EXPECT().Get(gomock.Any(), oauthtokens.ProviderTickTick).Return(token, nil)
	ticktick.EXPECT().CreateInboxTask(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, input googletasksync.CreateTickTickTaskInput) (googletasksync.TickTickTaskView, error) {
			if input.Title != "Refresh TickTick token" {
				t.Fatalf("unexpected title: %s", input.Title)
			}
			if input.Due != "" {
				t.Fatalf("expected no due date, got %s", input.Due)
			}
			if input.Priority != mediumPriority {
				t.Fatalf("unexpected priority: %d", input.Priority)
			}
			if !strings.Contains(input.Details, "Reconnect TickTick") {
				t.Fatalf("expected reconnect details, got %s", input.Details)
			}
			return googletasksync.TickTickTaskView{ID: "task-1"}, nil
		},
	)
	tokens.EXPECT().
		MarkRefreshReminderCreated(gomock.Any(), oauthtokens.ProviderTickTick, "access-1", "task-1", reminderNow).
		Return(nil)

	uc := New(tokens, ticktick, WithNow(func() time.Time { return reminderNow }))
	if err := uc.Handle(t.Context()); err != nil {
		t.Fatalf("handle reminder: %v", err)
	}
}

// Reports reminder creation failure without marking the token as reminded.
func TestHandleReportsReminderCreationFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	tokens := mocks.NewMockTokenRepository(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	token := tokenFixture()
	token.ExpiresAt = reminderNow.Add(24 * time.Hour)
	tokens.EXPECT().Get(gomock.Any(), oauthtokens.ProviderTickTick).Return(token, nil)
	ticktick.EXPECT().
		CreateInboxTask(gomock.Any(), gomock.Any()).
		Return(googletasksync.TickTickTaskView{}, errors.New("ticktick unavailable"))

	uc := New(tokens, ticktick, WithNow(func() time.Time { return reminderNow }))
	err := uc.Handle(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "ticktick unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Returns a default token with no expiry or reminder marker.
func tokenFixture() oauthtokens.Token {
	return oauthtokens.Token{AccessToken: "access-1"}
}
