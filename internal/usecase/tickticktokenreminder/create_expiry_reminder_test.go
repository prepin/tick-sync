package tickticktokenreminder_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/prepin/tick-sync/internal/infra/sqlite/oauthtokens"
	"github.com/prepin/tick-sync/internal/usecase/googletasksync"
	"github.com/prepin/tick-sync/internal/usecase/tickticktokenreminder"
	"github.com/prepin/tick-sync/internal/usecase/tickticktokenreminder/mocks"
)

const reminderTimeLayout = "2006-01-02 15:04:05 MST"

// Schedules Google reminders at the next local noon, including date and daylight-saving boundaries.
func TestCreateExpiryReminderSelectsNextLocalNoon(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		zone     string
		now      string
		wantNoon string
	}{
		{name: "before noon uses today", zone: "UTC", now: "2026-06-17T11:59:59Z", wantNoon: "2026-06-17T12:00:00Z"},
		{name: "exact noon uses today", zone: "UTC", now: "2026-06-17T12:00:00Z", wantNoon: "2026-06-17T12:00:00Z"},
		{name: "after noon uses tomorrow", zone: "UTC", now: "2026-06-17T12:00:00.000000001Z", wantNoon: "2026-06-18T12:00:00Z"},
		{name: "year boundary", zone: "UTC", now: "2026-12-31T13:00:00Z", wantNoon: "2027-01-01T12:00:00Z"},
		{name: "local calendar ahead of UTC", zone: "Pacific/Kiritimati", now: "2026-06-17T23:00:00Z", wantNoon: "2026-06-19T12:00:00+14:00"},
		{name: "local calendar behind UTC", zone: "America/Los_Angeles", now: "2026-06-18T01:00:00Z", wantNoon: "2026-06-18T12:00:00-07:00"},
		{name: "daylight saving starts overnight", zone: "Europe/Warsaw", now: "2026-03-28T13:00:00+01:00", wantNoon: "2026-03-29T12:00:00+02:00"},
		{name: "daylight saving ends overnight", zone: "Europe/Warsaw", now: "2026-10-24T13:00:00+02:00", wantNoon: "2026-10-25T12:00:00+01:00"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			location, err := time.LoadLocation(tc.zone)
			if err != nil {
				t.Fatalf("load timezone: %v", err)
			}
			now, err := time.Parse(time.RFC3339Nano, tc.now)
			if err != nil {
				t.Fatalf("parse current time: %v", err)
			}
			wantNoon, err := time.Parse(time.RFC3339, tc.wantNoon)
			if err != nil {
				t.Fatalf("parse expected noon: %v", err)
			}
			wantNoon = wantNoon.In(location)
			ctrl := gomock.NewController(t)
			tokens := mocks.NewMockTokenRepository(ctrl)
			google := mocks.NewMockGoogleTasksGateway(ctrl)
			token := tokenFixture()
			token.ExpiresAt = now.Add(-time.Hour)
			tokens.EXPECT().Get(gomock.Any(), oauthtokens.ProviderTickTick).Return(token, nil)
			google.EXPECT().CreateTask(gomock.Any(), tickticktokenreminder.CreateGoogleTaskInput{
				Title: "Reconnect TickTick — token expired",
				Notes: fmt.Sprintf(
					"TickTick token expired at %s. Reconnect TickTick at http://localhost:8080/.\n"+
						"Intended due time: %s (%s). Google Tasks API stores the date only.\n%s",
					token.ExpiresAt.In(location).Format(reminderTimeLayout),
					wantNoon.Format(reminderTimeLayout), tc.zone, googletasksync.TickTickExpiryReminderMarker,
				),
				Due: wantNoon,
			}).Return("google-reminder", nil)
			tokens.EXPECT().MarkExpiryReminderCreated(
				gomock.Any(), oauthtokens.ProviderTickTick, token.AccessToken, "google-reminder", now.In(location),
			).Return(nil)
			uc := tickticktokenreminder.New(
				tokens,
				mocks.NewMockTickTickGateway(ctrl),
				google,
				tickticktokenreminder.WithNow(
					func() time.Time { return now },
				),
				tickticktokenreminder.WithLocation(location),
			)
			if err := uc.Handle(t.Context()); err != nil {
				t.Fatalf("create expiry reminder: %v", err)
			}
		})
	}
}

// Retries a failed Google creation on the next check without marking the first attempt as delivered.
func TestCreateExpiryReminderRetriesFailedGoogleCreation(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	tokens := mocks.NewMockTokenRepository(ctrl)
	google := mocks.NewMockGoogleTasksGateway(ctrl)
	token := tokenFixture()
	token.ExpiresAt = reminderNow
	tokens.EXPECT().Get(gomock.Any(), oauthtokens.ProviderTickTick).Return(token, nil).Times(2)
	googleErr := errors.New("Google authentication unavailable")
	gomock.InOrder(
		google.EXPECT().CreateTask(gomock.Any(), gomock.Any()).Return("", googleErr),
		google.EXPECT().CreateTask(gomock.Any(), gomock.Any()).Return("google-reminder", nil),
		tokens.EXPECT().MarkExpiryReminderCreated(
			gomock.Any(), oauthtokens.ProviderTickTick, token.AccessToken, "google-reminder", reminderNow,
		).Return(nil),
	)
	uc := tickticktokenreminder.New(
		tokens,
		mocks.NewMockTickTickGateway(ctrl),
		google,
		tickticktokenreminder.WithNow(
			func() time.Time { return reminderNow },
		),
		tickticktokenreminder.WithLocation(time.UTC),
	)
	if err := uc.Handle(t.Context()); !errors.Is(err, googleErr) {
		t.Fatalf("expected Google failure, got %v", err)
	}
	if err := uc.Handle(t.Context()); err != nil {
		t.Fatalf("retry creation: %v", err)
	}
}

// Reports failure to persist the reminder marker instead of claiming successful delivery tracking.
func TestCreateExpiryReminderReportsUnrecordedDelivery(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	tokens := mocks.NewMockTokenRepository(ctrl)
	google := mocks.NewMockGoogleTasksGateway(ctrl)
	token := tokenFixture()
	token.ExpiresAt = reminderNow
	tokens.EXPECT().Get(gomock.Any(), oauthtokens.ProviderTickTick).Return(token, nil)
	storageErr := errors.New("storage unavailable")
	google.EXPECT().CreateTask(gomock.Any(), gomock.Any()).Return("google-reminder", nil)
	tokens.EXPECT().MarkExpiryReminderCreated(
		gomock.Any(), oauthtokens.ProviderTickTick, token.AccessToken, "google-reminder", reminderNow,
	).Return(storageErr)
	uc := tickticktokenreminder.New(
		tokens,
		mocks.NewMockTickTickGateway(ctrl),
		google,
		tickticktokenreminder.WithNow(
			func() time.Time { return reminderNow },
		),
		tickticktokenreminder.WithLocation(time.UTC),
	)
	if err := uc.Handle(t.Context()); !errors.Is(err, storageErr) {
		t.Fatalf("expected storage failure, got %v", err)
	}
}
