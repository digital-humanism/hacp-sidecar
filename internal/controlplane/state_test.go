package controlplane

import (
	"errors"
	"testing"
	"time"
)

func TestControlStateStartsStale(t *testing.T) {
	state := NewControlState(5 * time.Second)

	now := time.Unix(1000, 0)

	if state.IsFresh(now) {
		t.Fatal("new control state unexpectedly fresh")
	}
}

func TestControlStateSnapshotEstablishesFreshState(t *testing.T) {
	state := NewControlState(5 * time.Second)

	now := time.Unix(1000, 0)

	state.MarkSnapshot(42, now)

	if got := state.LastSeenRevision(); got != 42 {
		t.Fatalf(
			"last_seen_revision = %d, want 42",
			got,
		)
	}

	if !state.IsFresh(now) {
		t.Fatal("snapshot did not establish fresh state")
	}
}

func TestControlStateEventAdvancesRevision(t *testing.T) {
	state := NewControlState(5 * time.Second)

	t0 := time.Unix(1000, 0)
	t1 := t0.Add(time.Second)

	state.MarkSnapshot(10, t0)
	state.MarkEvent(11, t1)

	if got := state.LastSeenRevision(); got != 11 {
		t.Fatalf(
			"last_seen_revision = %d, want 11",
			got,
		)
	}

	if !state.LastUpdate().Equal(t1) {
		t.Fatalf(
			"last_update = %v, want %v",
			state.LastUpdate(),
			t1,
		)
	}
}

func TestControlStateDisconnectDoesNotImmediatelyMakeStateStale(
	t *testing.T,
) {
	state := NewControlState(5 * time.Second)

	t0 := time.Unix(1000, 0)

	state.MarkSnapshot(7, t0)
	state.MarkConnected()
	state.MarkDisconnected()

	if state.Connected() {
		t.Fatal("state unexpectedly connected")
	}

	// Still inside max staleness window.
	if !state.IsFresh(t0.Add(4 * time.Second)) {
		t.Fatal(
			"recent synchronized state became stale immediately after disconnect",
		)
	}
}

func TestControlStateFailsClosedAfterMaxStaleness(t *testing.T) {
	state := NewControlState(5 * time.Second)

	t0 := time.Unix(1000, 0)

	state.MarkSnapshot(7, t0)
	state.MarkDisconnected()

	if !state.IsFresh(t0.Add(5 * time.Second)) {
		t.Fatal(
			"state at exact max_staleness boundary should still be fresh",
		)
	}

	if state.IsFresh(t0.Add(5*time.Second + time.Nanosecond)) {
		t.Fatal(
			"state beyond max_staleness unexpectedly remained fresh",
		)
	}
}

func TestControlStateHeartbeatRefreshesFreshness(t *testing.T) {
	state := NewControlState(5 * time.Second)

	t0 := time.Unix(1000, 0)
	t1 := t0.Add(4 * time.Second)

	state.MarkSnapshot(15, t0)

	if err := state.MarkHeartbeat(15, t1); err != nil {
		t.Fatalf(
			"MarkHeartbeat: %v",
			err,
		)
	}

	// Without heartbeat this point would be stale relative to t0.
	// With heartbeat at t1 it remains fresh.
	now := t0.Add(8 * time.Second)

	if !state.IsFresh(now) {
		t.Fatal(
			"valid heartbeat did not refresh control-state freshness",
		)
	}

	if got := state.LastSeenRevision(); got != 15 {
		t.Fatalf(
			"heartbeat advanced revision to %d, want 15",
			got,
		)
	}
}

func TestControlStateHeartbeatRevisionMismatchFailsClosed(
	t *testing.T,
) {
	state := NewControlState(5 * time.Second)

	t0 := time.Unix(1000, 0)

	state.MarkSnapshot(15, t0)

	err := state.MarkHeartbeat(
		16,
		t0.Add(time.Second),
	)

	if err == nil {
		t.Fatal(
			"heartbeat revision mismatch unexpectedly succeeded",
		)
	}

	if !errors.Is(
		err,
		ErrHeartbeatRevisionMismatch,
	) {
		t.Fatalf(
			"error = %v, want ErrHeartbeatRevisionMismatch",
			err,
		)
	}

	if !state.Unsafe() {
		t.Fatal(
			"revision mismatch did not mark control state unsafe",
		)
	}

	if state.IsFresh(t0.Add(2 * time.Second)) {
		t.Fatal(
			"unsafe control state unexpectedly reported fresh",
		)
	}

	if got := state.LastSeenRevision(); got != 15 {
		t.Fatalf(
			"heartbeat mismatch changed revision to %d, want 15",
			got,
		)
	}
}

func TestControlStateExplicitUnsafeFailsClosedImmediately(
	t *testing.T,
) {
	state := NewControlState(5 * time.Second)

	t0 := time.Unix(1000, 0)

	state.MarkSnapshot(20, t0)

	if !state.IsFresh(t0) {
		t.Fatal("snapshot state unexpectedly stale")
	}

	state.MarkUnsafe()

	if state.IsFresh(t0) {
		t.Fatal(
			"explicit unsafe state unexpectedly remained fresh",
		)
	}
}

func TestControlStateValidSnapshotRecoversUnsafeState(
	t *testing.T,
) {
	state := NewControlState(5 * time.Second)

	t0 := time.Unix(1000, 0)
	t1 := t0.Add(time.Second)

	state.MarkSnapshot(20, t0)
	state.MarkUnsafe()

	if state.IsFresh(t0) {
		t.Fatal("unsafe state unexpectedly fresh")
	}

	// A complete atomic snapshot re-establishes trusted state.
	state.MarkSnapshot(25, t1)

	if state.Unsafe() {
		t.Fatal(
			"valid snapshot did not clear unsafe state",
		)
	}

	if !state.IsFresh(t1) {
		t.Fatal(
			"snapshot recovery did not restore freshness",
		)
	}

	if got := state.LastSeenRevision(); got != 25 {
		t.Fatalf(
			"last_seen_revision = %d, want 25",
			got,
		)
	}
}
