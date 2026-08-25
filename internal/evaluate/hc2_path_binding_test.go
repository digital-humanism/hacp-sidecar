package evaluate

import (
	"testing"

	"hacp-sidecar/internal/wire"
)

func TestHC2PercentTripletHexCaseEquivalence(t *testing.T) {
	guard := NewDefaultScopeGuard()

	constraints := &wire.Constraints{
		Path: "/a%2Fb",
	}

	req := &RequestContext{
		Path: "/a%2fb",
	}

	if !guard.MatchRequestConstraints(constraints, req) {
		t.Fatalf(
			"HC2 percent-triplet hex case equivalence rejected: constraint path=%q request path=%q",
			constraints.Path,
			req.Path,
		)
	}
}
