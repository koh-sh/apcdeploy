package batch

import "sync"

// PayloadCollector accumulates per-target stdout payloads (plus a
// hasChanges flag and an optional post-run warning) from goroutines
// launched by Orchestrator, and exposes them in argument order
// regardless of completion order.
//
// It exists because Orchestrator's worker pool finishes targets in
// completion order, but commands like `diff` need a deterministic,
// argument-ordered output stream. Without a shared collector, each
// caller would have to repeat the index-map + mutex + slot-based
// slice plumbing — keeping that detail out of cmd/ is the whole
// point of this type.
//
// The warning slot exists for notices that must not be written while
// the Orchestrator's Targets block is still open (its redraw would
// overwrite them): callers record them here and emit them after
// Orchestrator.Run returns.
//
// The zero value is not usable; use NewPayloadCollector. Set is
// goroutine-safe; Payloads / HasChanges / Warnings return shared slices and
// MUST only be called after the orchestrator has finished
// (i.e. after Orchestrator.Run returns).
type PayloadCollector struct {
	payloads   [][]byte
	hasChanges []bool
	warnings   []string
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
		warnings:   make([]string, len(targets)),
		indexByID:  make(map[string]int, len(targets)),
	}
	for i, t := range targets {
		pc.indexByID[t.Identifier] = i
	}
	return pc
}

// Set records the payload, hasChanges flag, and warning ("" for none)
// for the target whose canonical identifier matches. Calls for unknown identifiers are
// silently ignored — the executor surface can be extended without
// teaching every collector about new targets, and collectors built
// for a different batch never accidentally mutate state owned by
// another run.
func (pc *PayloadCollector) Set(identifier string, payload []byte, hasChanges bool, warning string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	idx, ok := pc.indexByID[identifier]
	if !ok {
		return
	}
	pc.payloads[idx] = payload
	pc.hasChanges[idx] = hasChanges
	pc.warnings[idx] = warning
}

// Payloads returns the per-target payload slice in argument order.
// Slot i corresponds to the i-th Target supplied to NewPayloadCollector.
// A nil slot means the target produced no payload (no-op or failed).
func (pc *PayloadCollector) Payloads() [][]byte { return pc.payloads }

// HasChanges returns the per-target hasChanges flags in argument order,
// aligned with Payloads. Used by command-level policies such as
// `diff --exit-nonzero` that collapse "any change" across all targets.
func (pc *PayloadCollector) HasChanges() []bool { return pc.hasChanges }

// Warnings returns the per-target warnings in argument order, aligned
// with Payloads. An empty slot means the target recorded no warning.
func (pc *PayloadCollector) Warnings() []string { return pc.warnings }
