package testing

import (
	"strings"
	"sync"
	"time"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// TargetsCall captures the lifecycle of a single Targets invocation: the
// initial identifier list passed to Reporter.Targets, every state transition
// in the order it was issued, and whether Close was called.
type TargetsCall struct {
	IDs         []string
	Transitions []TargetsTransition
	Closed      bool
}

// TargetsTransition records one method call against a Targets handle. Kind
// is one of "phase", "progress", "done", "fail", or "skip".
type TargetsTransition struct {
	Kind    string
	ID      string
	Phase   string
	Detail  string
	Percent float64
	ETA     time.Duration
	Summary string
	Reason  string
	Err     error
}

// Targets opens a new Targets handle. The returned handle records every
// transition into the matching TargetsCall entry.
//
// The returned handle is safe for concurrent calls — the multi-config
// orchestrator (internal/batch) drives several targets in parallel from
// separate goroutines, and tests of that path need a race-free mock.
func (m *MockReporter) Targets(ids []string) reporter.Targets {
	idx := len(m.TargetsCalls)
	m.TargetsCalls = append(m.TargetsCalls, TargetsCall{
		IDs: append([]string(nil), ids...),
	})
	m.Messages = append(m.Messages, "targets: "+strings.Join(ids, ","))
	return &mockTargets{m: m, idx: idx}
}

type mockTargets struct {
	m      *MockReporter
	idx    int
	mu     sync.Mutex
	closed bool
}

func (t *mockTargets) SetPhase(id, phase, detail string) {
	t.record(TargetsTransition{Kind: "phase", ID: id, Phase: phase, Detail: detail})
}

func (t *mockTargets) SetProgress(id string, percent float64, eta time.Duration) {
	t.record(TargetsTransition{Kind: "progress", ID: id, Percent: percent, ETA: eta})
}

func (t *mockTargets) Done(id, summary string) {
	t.record(TargetsTransition{Kind: "done", ID: id, Summary: summary})
}

func (t *mockTargets) Fail(id string, err error) {
	t.record(TargetsTransition{Kind: "fail", ID: id, Err: err})
}

func (t *mockTargets) Skip(id, reason string) {
	t.record(TargetsTransition{Kind: "skip", ID: id, Reason: reason})
}

func (t *mockTargets) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	t.closed = true
	t.m.TargetsCalls[t.idx].Closed = true
	t.m.Messages = append(t.m.Messages, "targets-close")
}

func (t *mockTargets) record(tr TargetsTransition) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	t.m.TargetsCalls[t.idx].Transitions = append(t.m.TargetsCalls[t.idx].Transitions, tr)
	t.m.Messages = append(t.m.Messages, "targets-"+tr.Kind+": "+tr.ID)
}
