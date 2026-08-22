package trust

import "errors"

var (
	ErrInvalidKeyID          = errors.New("invalid key id")
	ErrInvalidPublicKey      = errors.New("invalid Ed25519 public key")
	ErrConflictingKeyBinding = errors.New("conflicting key binding")
	ErrEmptyTrustSnapshot    = errors.New("empty trust snapshot")
	ErrInvalidTrustSnapshot  = errors.New("invalid trust snapshot")
	ErrTrustNotReady         = errors.New("trust store not ready")
	ErrKeyNotFound           = errors.New("key not found")
	ErrTrustRevisionRollback = errors.New("trust revision rollback")
	ErrTrustRevisionConflict = errors.New("trust revision conflict")
)
