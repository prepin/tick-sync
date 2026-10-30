package tickticktokenreminder_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/prepin/tick-sync/internal/infra/sqlite/oauthtokens"
	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
	"github.com/prepin/tick-sync/internal/usecase/tickticktokenreminder"
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

	uc := tickticktokenreminder.New(
		tokens,
		ticktick,
		mocks.NewMockGoogleTasksGateway(ctrl),
		tickticktokenreminder.WithNow(func() time.Time { return reminderNow }),
	)
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

	uc := tickticktokenreminder.New(
		tokens,
		ticktick,
		mocks.NewMockGoogleTasksGateway(ctrl),
		tickticktokenreminder.WithNow(func() time.Time { return reminderNow }),
	)
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

	uc := tickticktokenreminder.New(
		tokens,
		ticktick,
		mocks.NewMockGoogleTasksGateway(ctrl),
		tickticktokenreminder.WithNow(func() time.Time { return reminderNow }),
	)
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

	uc := tickticktokenreminder.New(
		tokens,
		ticktick,
		mocks.NewMockGoogleTasksGateway(ctrl),
		tickticktokenreminder.WithNow(func() time.Time { return reminderNow }),
	)
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
			if input.Priority != 3 {
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

	uc := tickticktokenreminder.New(
		tokens,
		ticktick,
		mocks.NewMockGoogleTasksGateway(ctrl),
		tickticktokenreminder.WithNow(func() time.Time { return reminderNow }),
	)
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

	uc := tickticktokenreminder.New(
		tokens,
		ticktick,
		mocks.NewMockGoogleTasksGateway(ctrl),
		tickticktokenreminder.WithNow(func() time.Time { return reminderNow }),
	)
	err := uc.Handle(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "ticktick unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Creates the Google reminder at or after expiry even if a pre-expiry TickTick reminder already exists.
func TestHandleCreatesGoogleReminderForExpiredToken(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		expiresAt time.Time
	}{
		{name: "token expires exactly now", expiresAt: reminderNow},
		{name: "token already expired", expiresAt: reminderNow.Add(-time.Hour)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			tokens := mocks.NewMockTokenRepository(ctrl)
			ticktick := mocks.NewMockTickTickGateway(ctrl)
			google := mocks.NewMockGoogleTasksGateway(ctrl)
			token := tokenFixture()
			token.ExpiresAt = tc.expiresAt
			token.RefreshReminderTaskID = "pre-expiry-reminder"
			tokens.EXPECT().Get(gomock.Any(), oauthtokens.ProviderTickTick).Return(token, nil)
			google.EXPECT().CreateTask(gomock.Any(), gomock.Any()).Return("google-reminder", nil)
			tokens.EXPECT().MarkExpiryReminderCreated(
				gomock.Any(), oauthtokens.ProviderTickTick, token.AccessToken, "google-reminder", reminderNow,
			).Return(nil)
			uc := tickticktokenreminder.New(
				tokens,
				ticktick,
				google,
				tickticktokenreminder.WithNow(func() time.Time { return reminderNow }),
				tickticktokenreminder.WithLocation(time.UTC),
			)
			if err := uc.Handle(t.Context()); err != nil {
				t.Fatalf("handle expired token: %v", err)
			}
		})
	}
}

// Leaves an already-recorded Google expiry reminder untouched without contacting either task API.
func TestHandleDoesNotDuplicateGoogleExpiryReminder(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	tokens := mocks.NewMockTokenRepository(ctrl)
	token := tokenFixture()
	token.ExpiresAt = reminderNow.Add(-time.Hour)
	token.ExpiryReminderTaskID = "google-reminder"
	tokens.EXPECT().Get(gomock.Any(), oauthtokens.ProviderTickTick).Return(token, nil).Times(2)
	uc := tickticktokenreminder.New(tokens, mocks.NewMockTickTickGateway(ctrl), mocks.NewMockGoogleTasksGateway(ctrl),
		tickticktokenreminder.WithNow(func() time.Time { return reminderNow }))
	for range 2 {
		if err := uc.Handle(t.Context()); err != nil {
			t.Fatalf("handle expired token: %v", err)
		}
	}
}

// Reports unavailable token metadata without creating reminders in either provider.
func TestHandleReportsUnavailableTokenMetadata(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	tokens := mocks.NewMockTokenRepository(ctrl)
	storageErr := errors.New("storage unavailable")
	tokens.EXPECT().Get(gomock.Any(), oauthtokens.ProviderTickTick).Return(oauthtokens.Token{}, storageErr)
	uc := tickticktokenreminder.New(tokens, mocks.NewMockTickTickGateway(ctrl), mocks.NewMockGoogleTasksGateway(ctrl))
	if err := uc.Handle(t.Context()); !errors.Is(err, storageErr) {
		t.Fatalf("expected storage failure, got %v", err)
	}
}

// Returns a default token with no expiry or reminder marker.
func tokenFixture() oauthtokens.Token {
	return oauthtokens.Token{AccessToken: "access-1"}
}
