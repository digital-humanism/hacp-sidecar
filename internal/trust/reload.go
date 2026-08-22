package trust

import "fmt"

// ReloadFromFile loads a complete candidate trust snapshot from path and
// atomically replaces the active snapshot only after full validation.
//
// Invalid, conflicting, stale, or rolled-back candidates leave the current
// active trust state unchanged.
func ReloadFromFile(store *AtomicTrustStore, path string) error {
	if store == nil {
		return fmt.Errorf("%w: nil trust store", ErrTrustNotReady)
	}

	snapshot, err := LoadSnapshotFile(path)
	if err != nil {
		return err
	}
	return store.Replace(snapshot)
}
