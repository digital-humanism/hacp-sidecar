package controlplane

import (
	"context"
	"errors"
	"time"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server exposes Journal through the HACP control-plane gRPC contract.
type Server struct {
	controlplanev1.UnimplementedControlPlaneServer

	journal *Journal

	// Zero means heartbeat delivery is disabled.
	//
	// This keeps existing tests deterministic while allowing production
	// runtime and heartbeat integration tests to enable it explicitly.
	heartbeatInterval time.Duration

	now func() time.Time
}

// NewServer creates a control-plane service without periodic heartbeats.
//
// Heartbeat-enabled runtime wiring should use NewServerWithHeartbeat.
func NewServer(journal *Journal) *Server {
	return newServer(
		journal,
		0,
		time.Now,
	)
}

// NewServerWithHeartbeat creates a control-plane service that emits periodic
// heartbeat messages while a WatchRevocations stream is caught up.
func NewServerWithHeartbeat(
	journal *Journal,
	heartbeatInterval time.Duration,
) *Server {
	if heartbeatInterval <= 0 {
		panic("controlplane: heartbeat interval must be positive")
	}

	return newServer(
		journal,
		heartbeatInterval,
		time.Now,
	)
}

func newServer(
	journal *Journal,
	heartbeatInterval time.Duration,
	now func() time.Time,
) *Server {
	if journal == nil {
		panic("controlplane: nil journal")
	}

	if now == nil {
		panic("controlplane: nil clock")
	}

	return &Server{
		journal:           journal,
		heartbeatInterval: heartbeatInterval,
		now:               now,
	}
}

// GetRevocationSnapshot returns complete revocation state at one atomic
// control-plane revision.
func (s *Server) GetRevocationSnapshot(
	_ context.Context,
	_ *controlplanev1.GetRevocationSnapshotRequest,
) (*controlplanev1.RevocationSnapshot, error) {
	return s.journal.Snapshot(), nil
}

// WatchRevocations:
//
//   - replays revisions > after_revision
//   - continues with live revisions
//   - returns ResetRequired if replay is no longer possible
//   - emits heartbeat while the client is fully caught up
//
// sequence is stream-local.
// revision remains the only durable recovery cursor.
func (s *Server) WatchRevocations(
	req *controlplanev1.WatchRevocationsRequest,
	stream grpc.ServerStreamingServer[controlplanev1.WatchRevocationsResponse],
) error {
	if req == nil {
		return status.Error(
			codes.InvalidArgument,
			"watch revocations request is required",
		)
	}

	lastRevision := req.GetAfterRevision()

	var sequence uint64

	for {
		events, notify, err := s.journal.EventsAfter(lastRevision)
		if err != nil {
			if errors.Is(err, ErrReplayUnavailable) {
				sequence++

				if sendErr := stream.Send(
					&controlplanev1.WatchRevocationsResponse{
						Sequence: sequence,
						Payload: &controlplanev1.WatchRevocationsResponse_ResetRequired{
							ResetRequired: &controlplanev1.ResetRequired{
								OldestAvailableRevision: s.journal.OldestAvailableRevision(),
								CurrentRevision:         s.journal.Revision(),
								Reason:                  "requested revision is outside replay retention",
							},
						},
					},
				); sendErr != nil {
					return sendErr
				}

				return nil
			}

			if errors.Is(err, ErrRevisionAhead) {
				return status.Error(
					codes.FailedPrecondition,
					err.Error(),
				)
			}

			return status.Error(
				codes.Internal,
				err.Error(),
			)
		}

		// -------------------------------------------------------------
		// Replay / live events always take priority over heartbeat.
		// -------------------------------------------------------------

		if len(events) > 0 {
			for _, event := range events {
				sequence++

				response := &controlplanev1.WatchRevocationsResponse{
					Sequence: sequence,
					Payload: &controlplanev1.WatchRevocationsResponse_Event{
						Event: event,
					},
				}

				if err := stream.Send(response); err != nil {
					return err
				}

				lastRevision = event.GetRevision()
			}

			continue
		}

		// -------------------------------------------------------------
		// Heartbeat disabled.
		// -------------------------------------------------------------

		if s.heartbeatInterval <= 0 {
			select {
			case <-stream.Context().Done():
				return stream.Context().Err()

			case <-notify:
				// A new revision was committed.
				continue
			}
		}

		// -------------------------------------------------------------
		// Heartbeat enabled.
		// -------------------------------------------------------------

		timer := time.NewTimer(s.heartbeatInterval)

		select {
		case <-stream.Context().Done():
			stopTimer(timer)
			return stream.Context().Err()

		case <-notify:
			// A new event exists. Do not send heartbeat before the event.
			stopTimer(timer)
			continue

		case <-timer.C:
			// IMPORTANT:
			//
			// A revision may have been committed at almost the same instant
			// as the heartbeat timer fired.
			//
			// If the journal advanced, loop back and deliver the event first.
			// Otherwise we could send:
			//
			//   heartbeat current_revision=11
			//
			// to a sidecar that has only materialized revision 10.
			//
			// That would incorrectly look like a protocol violation.
			currentRevision := s.journal.Revision()

			if currentRevision != lastRevision {
				continue
			}

			sequence++

			response := &controlplanev1.WatchRevocationsResponse{
				Sequence: sequence,
				Payload: &controlplanev1.WatchRevocationsResponse_Heartbeat{
					Heartbeat: &controlplanev1.Heartbeat{
						CurrentRevision: lastRevision,
						ServerTimeMs:    s.now().UnixMilli(),
					},
				},
			}

			if err := stream.Send(response); err != nil {
				return err
			}
		}
	}
}

func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}

	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}
