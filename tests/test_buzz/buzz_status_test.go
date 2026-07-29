package test_buzz

import (
	"testing"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	buzzservice "github.com/hngprojects/telex_be/services/buzz"
)

func TestDetermineBuzzStatus(t *testing.T) {
	now := time.Now().UTC()
	futureTime := now.Add(1 * time.Hour)
	pastTime := now.Add(-1 * time.Hour)

	t.Run("StatusAlreadyEnded", func(t *testing.T) {
		b := &models.Buzz{
			Status:      models.BuzzStatusEnded,
			BuzzEndTime: &futureTime,
		}
		if got := buzzservice.DetermineBuzzStatus(b); got != models.BuzzStatusEnded {
			t.Errorf("expected %s, got %s", models.BuzzStatusEnded, got)
		}
	})

	t.Run("EndTimeInFuture", func(t *testing.T) {
		b := &models.Buzz{
			Status:        models.BuzzStatusActive,
			BuzzStartTime: now.Add(-30 * time.Minute),
			BuzzEndTime:   &futureTime,
		}
		if got := buzzservice.DetermineBuzzStatus(b); got != models.BuzzStatusActive {
			t.Errorf("expected %s, got %s", models.BuzzStatusActive, got)
		}
	})

	t.Run("EndTimeInPast", func(t *testing.T) {
		b := &models.Buzz{
			Status:        models.BuzzStatusActive,
			BuzzStartTime: now.Add(-3 * time.Hour),
			BuzzEndTime:   &pastTime,
		}
		if got := buzzservice.DetermineBuzzStatus(b); got != models.BuzzStatusEnded {
			t.Errorf("expected %s, got %s", models.BuzzStatusEnded, got)
		}
	})

	t.Run("NilEndTimeWithinCap", func(t *testing.T) {
		b := &models.Buzz{
			Status:        models.BuzzStatusActive,
			BuzzStartTime: now.Add(-30 * time.Minute),
			BuzzEndTime:   nil,
		}
		if got := buzzservice.DetermineBuzzStatus(b); got != models.BuzzStatusActive {
			t.Errorf("expected %s, got %s", models.BuzzStatusActive, got)
		}
	})

	t.Run("NilEndTimeExceedsCap", func(t *testing.T) {
		b := &models.Buzz{
			Status:        models.BuzzStatusActive,
			BuzzStartTime: now.Add(-3 * time.Hour),
			BuzzEndTime:   nil,
		}
		if got := buzzservice.DetermineBuzzStatus(b); got != models.BuzzStatusEnded {
			t.Errorf("expected %s, got %s", models.BuzzStatusEnded, got)
		}
	})
}
