package controlplane

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"
)

var (
	ErrInvalidRevocationKind = errors.New("invalid revocation kind")
	ErrEmptySubjectID        = errors.New("empty revocation subject id")
	ErrRevisionAhead         = errors.New("requested revision is ahead of control-plane revision")
	ErrReplayUnavailable     = errors.New("requested revision is no longer available for replay")
)

// clockFunc exists so tests can make timestamps deterministic.
type clockFunc func() time.Time

// Journal is the authoritative in-memory revocation journal.
//
// Properties:
//
//   - revision is globally monotonic inside this control-plane instance.
//   - revocation state is monotonic: entries may be added but not removed.
//   - duplicate revocation of the same (kind, subject_id) is idempotent.
//   - events contains the full history for replay.
//   - notify is closed whenever a new revision is committed.
//
// Gate E2 deliberately keeps the complete history in memory.
// Retention / compaction / ResetRequired are added in the recovery phase.
type Journal struct {
	mu sync.RWMutex

	revision uint64
	events   []*controlplanev1.RevocationEvent

	revoked map[controlplanev1.RevocationKind]map[string]struct{}

	// notify represents "something changed after the state you observed".
	//
	// Each commit closes the current channel and replaces it with a new one.
	// Watchers therefore cannot miss a transition between replay and waiting.
	notify chan struct{}

	now clockFunc
}

// NewJournal creates an empty control-plane journal at revision 0.
func NewJournal() *Journal {
	return newJournalWithClock(time.Now)
}

func newJournalWithClock(now clockFunc) *Journal {
	return &Journal{
		revoked: make(
			map[controlplanev1.RevocationKind]map[string]struct{},
		),
		notify: make(chan struct{}),
		now:    now,
	}
}

// Revision returns the highest committed revision.
func (j *Journal) Revision() uint64 {
	j.mu.RLock()
	defer j.mu.RUnlock()

	return j.revision
}

// Revoke commits a revocation.
//
// Returns:
//
//	event   - committed event, or nil for an idempotent duplicate
//	created - true only when a new control-plane revision was created
//
// Duplicate revocation is intentionally a no-op. Since the distributed
// revocation state did not change, it does not consume another revision.
func (j *Journal) Revoke(
	kind controlplanev1.RevocationKind,
	subjectID string,
) (*controlplanev1.RevocationEvent, bool, error) {
	if err := validateRevocation(kind, subjectID); err != nil {
		return nil, false, err
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	set := j.revoked[kind]
	if set == nil {
		set = make(map[string]struct{})
		j.revoked[kind] = set
	}

	// Idempotent duplicate.
	if _, exists := set[subjectID]; exists {
		return nil, false, nil
	}

	j.revision++

	event := &controlplanev1.RevocationEvent{
		Revision:   j.revision,
		EventId:    fmt.Sprintf("revocation-%020d", j.revision),
		Kind:       kind,
		SubjectId:  subjectID,
		IssuedAtMs: j.now().UnixMilli(),
	}

	// Commit snapshot state first.
	set[subjectID] = struct{}{}

	// Then append the immutable event to replay history.
	j.events = append(j.events, cloneEvent(event))

	// Wake every WatchRevocations stream.
	close(j.notify)
	j.notify = make(chan struct{})

	return cloneEvent(event), true, nil
}

// Snapshot returns the complete revocation state at one atomic revision.
func (j *Journal) Snapshot() *controlplanev1.RevocationSnapshot {
	j.mu.RLock()
	defer j.mu.RUnlock()

	entries := make([]*controlplanev1.RevocationEntry, 0)

	for kind, subjects := range j.revoked {
		for subjectID := range subjects {
			entries = append(entries, &controlplanev1.RevocationEntry{
				Kind:      kind,
				SubjectId: subjectID,
			})
		}
	}

	// Deterministic ordering makes integration tests and diagnostics stable.
	sort.Slice(entries, func(a, b int) bool {
		if entries[a].Kind != entries[b].Kind {
			return entries[a].Kind < entries[b].Kind
		}

		return entries[a].SubjectId < entries[b].SubjectId
	})

	return &controlplanev1.RevocationSnapshot{
		Revision:      j.revision,
		Entries:       entries,
		GeneratedAtMs: j.now().UnixMilli(),
	}
}

// EventsAfter atomically returns:
//
//   - all events with revision > afterRevision
//   - a notification channel representing the state observed
//
// If no events are currently available, a WatchRevocations caller can wait on
// the returned channel. Any later commit closes that exact channel.
//
// This prevents the classic race:
//
//	read journal
//	event arrives
//	start waiting forever
//
// because journal observation and notification-channel acquisition happen
// under the same lock.
func (j *Journal) EventsAfter(
	afterRevision uint64,
) (
	[]*controlplanev1.RevocationEvent,
	<-chan struct{},
	error,
) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	if afterRevision > j.revision {
		return nil, nil, fmt.Errorf(
			"%w: after_revision=%d current_revision=%d",
			ErrRevisionAhead,
			afterRevision,
			j.revision,
		)
	}

	if len(j.events) > 0 {
		oldest := j.events[0].Revision

		// To replay oldest revision R, the client may resume from R-1.
		//
		// Anything older means at least one required event has already
		// disappeared from retention.
		if oldest > 0 && afterRevision < oldest-1 {
			return nil, nil, fmt.Errorf(
				"%w: after_revision=%d oldest_available_revision=%d current_revision=%d",
				ErrReplayUnavailable,
				afterRevision,
				oldest,
				j.revision,
			)
		}
	}

	if afterRevision == j.revision {
		return nil, j.notify, nil
	}

	events := make(
		[]*controlplanev1.RevocationEvent,
		0,
		j.revision-afterRevision,
	)

	// E2 keeps complete history and revisions start at 1.
	// Therefore events[index] corresponds to revision index+1.
	for _, event := range j.events {
		if event.Revision > afterRevision {
			events = append(events, cloneEvent(event))
		}
	}

	return events, j.notify, nil
}

func validateRevocation(
	kind controlplanev1.RevocationKind,
	subjectID string,
) error {
	if subjectID == "" {
		return ErrEmptySubjectID
	}

	switch kind {
	case controlplanev1.RevocationKind_REVOCATION_KIND_KEY,
		controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		controlplanev1.RevocationKind_REVOCATION_KIND_ENVELOPE,
		controlplanev1.RevocationKind_REVOCATION_KIND_PARENT_ENVELOPE:
		return nil

	default:
		return fmt.Errorf(
			"%w: %s",
			ErrInvalidRevocationKind,
			kind.String(),
		)
	}
}

func cloneEvent(
	event *controlplanev1.RevocationEvent,
) *controlplanev1.RevocationEvent {
	if event == nil {
		return nil
	}

	return &controlplanev1.RevocationEvent{
		Revision:   event.Revision,
		EventId:    event.EventId,
		Kind:       event.Kind,
		SubjectId:  event.SubjectId,
		IssuedAtMs: event.IssuedAtMs,
	}
}

// OldestAvailableRevision returns the earliest event revision still retained
// for replay.
//
// If the journal contains no replayable events, it returns revision + 1.
func (j *Journal) OldestAvailableRevision() uint64 {
	j.mu.RLock()
	defer j.mu.RUnlock()

	if len(j.events) == 0 {
		return j.revision + 1
	}

	return j.events[0].Revision
}

// CompactThrough removes replay history through the supplied revision.
//
// IMPORTANT:
//
// Revocation snapshot state is NOT removed. Only historical events used for
// incremental replay are compacted.
//
// Example:
//
// current revision = 5
// CompactThrough(3)
//
// snapshot still contains state from revisions 1..5
// replay history contains revisions 4..5
func (j *Journal) CompactThrough(revision uint64) {
	j.mu.Lock()
	defer j.mu.Unlock()

	idx := 0

	for idx < len(j.events) &&
		j.events[idx].Revision <= revision {
		idx++
	}

	if idx == 0 {
		return
	}

	remaining := make(
		[]*controlplanev1.RevocationEvent,
		len(j.events)-idx,
	)

	copy(remaining, j.events[idx:])

	j.events = remaining
}
