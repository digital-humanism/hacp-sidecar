package main

import (
	"testing"
	"time"
)

type readinessTestControlState struct {
	fresh       bool
	lastChecked time.Time
}

func (s *readinessTestControlState) IsFresh(
	now time.Time,
) bool {

	s.lastChecked = now
	return s.fresh
}

func TestControlStateReadinessFresh(
	t *testing.T,
) {

	now :=
		time.Date(
			2026,
			time.August,
			22,
			12,
			0,
			0,
			0,
			time.UTC,
		)

	state :=
		&readinessTestControlState{
			fresh: true,
		}

	ready :=
		makeControlStateReadiness(
			state,
			func() time.Time {
				return now
			},
		)

	if !ready() {
		t.Fatal(
			"expected fresh control state to be ready",
		)
	}

	if !state.lastChecked.Equal(now) {
		t.Fatalf(
			"expected readiness clock %v, got %v",
			now,
			state.lastChecked,
		)
	}
}

func TestControlStateReadinessStale(
	t *testing.T,
) {

	state :=
		&readinessTestControlState{
			fresh: false,
		}

	ready :=
		makeControlStateReadiness(
			state,
			time.Now,
		)

	if ready() {
		t.Fatal(
			"expected stale control state to be not ready",
		)
	}
}

func TestControlStateReadinessMissingStateFailsClosed(
	t *testing.T,
) {

	ready :=
		makeControlStateReadiness(
			nil,
			time.Now,
		)

	if ready() {
		t.Fatal(
			"expected missing control state to fail closed",
		)
	}
}

func TestControlStateReadinessMissingClockFailsClosed(
	t *testing.T,
) {

	state :=
		&readinessTestControlState{
			fresh: true,
		}

	ready :=
		makeControlStateReadiness(
			state,
			nil,
		)

	if ready() {
		t.Fatal(
			"expected missing readiness clock to fail closed",
		)
	}
}

func TestControlStateReadinessTracksRecovery(
	t *testing.T,
) {

	state :=
		&readinessTestControlState{
			fresh: false,
		}

	ready :=
		makeControlStateReadiness(
			state,
			time.Now,
		)

	if ready() {
		t.Fatal(
			"expected initial stale state to be not ready",
		)
	}

	state.fresh = true

	if !ready() {
		t.Fatal(
			"expected recovered fresh state to become ready",
		)
	}
}
