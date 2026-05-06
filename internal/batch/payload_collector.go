package batch

import "sync"

// PayloadCollector accumulates per-target stdout payloads (and a
// hasChanges flag) from goroutines launched by Orchestrator, and
// exposes them in argument order regardless of completion order.
//
// It exists because Orchestrator's worker pool finishes targets in
// completion order, but commands like `diff` need a deterministic,
// argument-ordered output stream. Without a shared collector, each
// caller would have to repeat the index-map + mutex + slot-based
// slice plumbing — keeping that detail out of cmd/ is the whole
// point of this type.
//
// The zero value is not usable; use NewPayloadCollector. Set is
// goroutine-safe; Payloads / HasChanges return shared slices and
// MUST only be called after the orchestrator has finished
// (i.e. after Orchestrator.Run returns).
type PayloadCollector struct {
	payloads   [][]byte
	hasChanges []bool
	indexByID  map[string]int
	mu         sync.Mutex
}

// NewPayloadCollector returns a collector sized for the given Targets.
// The argument-order slot for each target is determined at
// construction time, so Targets MUST NOT be reordered between this
// call and the orchestrator run.
func NewPayloadCollector(targets []*Target) *PayloadCollector {
	pc := &PayloadCollector{
		payloads:   make([][]byte, len(targets)),
		hasChanges: make([]bool, len(targets)),
		indexByID:  make(map[string]int, len(targets)),
	}
	for i, t := range targets {
		pc.indexByID[t.Identifier] = i
	}
	return pc
}

// Set records the payload and hasChanges flag for the target whose
// canonical identifier matches. Calls for unknown identifiers are
// silently ignored — the executor surface can be extended without
// teaching every collector about new targets, and collectors built
// for a different batch never accidentally mutate state owned by
// another run.
func (pc *PayloadCollector) Set(identifier string, payload []byte, hasChanges bool) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	idx, ok := pc.indexByID[identifier]
	if !ok {
		return
	}
	pc.payloads[idx] = payload
	pc.hasChanges[idx] = hasChanges
}

// Payloads returns the per-target payload slice in argument order.
// Slot i corresponds to the i-th Target supplied to NewPayloadCollector.
// A nil slot means the target produced no payload (no-op or failed).
func (pc *PayloadCollector) Payloads() [][]byte { return pc.payloads }

// HasChanges returns the per-target hasChanges flags in argument order,
// aligned with Payloads. Used by command-level policies such as
// `diff --exit-nonzero` that collapse "any change" across all targets.
func (pc *PayloadCollector) HasChanges() []bool { return pc.hasChanges }
