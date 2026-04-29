// Package testing provides test utilities for the reporter package.
package testing

import (
	"strings"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// MockReporter records every call made to a Reporter, enabling assertions in
// unit tests. Each invocation appends a tagged entry to Messages with the
// kind name as a prefix (e.g. "step: ..."). Stdout payloads are recorded as
// well so tests can inspect the bytes that would have been written.
type MockReporter struct {
	Messages []string
	Stdout   []byte

	// Tables records every Table invocation (headers + rows) so callers can
	// assert on structured output.
	Tables []TableCall
	// Boxes records every Box invocation (title + lines).
	Boxes []BoxCall

	// SpinnerCalls records each spinner lifecycle: the start message and the
	// terminating Done/Fail message.
	SpinnerCalls []SpinnerCall

	// ProgressCalls records each progress-bar lifecycle: the start message,
	// every Update call, and the terminating Done/Fail message.
	ProgressCalls []ProgressCall

	// ChecklistCalls records each checklist lifecycle: the initial item
	// labels and every state transition.
	ChecklistCalls []ChecklistCall
}

// ChecklistCall captures the lifecycle of a single Checklist invocation.
type ChecklistCall struct {
	Items       []string
	Transitions []ChecklistTransition
	Closed      bool
}

// ChecklistTransition records one Start/Done/Fail/Skip call on a checklist.
type ChecklistTransition struct {
	Index   int
	Outcome string // "start", "done", "fail", "skip"
	Message string
}

// ProgressCall captures the lifecycle of a single progress bar.
type ProgressCall struct {
	StartMessage string
	Updates      []ProgressUpdate
	Outcome      string // "done" or "fail"
	EndMessage   string
}

// ProgressUpdate captures a single Update invocation on a progress bar.
type ProgressUpdate struct {
	Percent float64
	Message string
}

// TableCall captures the arguments to Reporter.Table.
type TableCall struct {
	Headers []string
	Rows    [][]string
}

// BoxCall captures the arguments to Reporter.Box.
type BoxCall struct {
	Title string
	Lines []string
}

// SpinnerCall captures the lifecycle of a single spinner: the message passed
// to Spin, plus the terminating Done/Fail outcome and message.
type SpinnerCall struct {
	StartMessage string
	Outcome      string // "done" or "fail"
	EndMessage   string
}

var _ reporter.Reporter = (*MockReporter)(nil)

func (m *MockReporter) Step(msg string)    { m.Messages = append(m.Messages, "step: "+msg) }
func (m *MockReporter) Success(msg string) { m.Messages = append(m.Messages, "success: "+msg) }
func (m *MockReporter) Info(msg string)    { m.Messages = append(m.Messages, "info: "+msg) }
func (m *MockReporter) Warn(msg string)    { m.Messages = append(m.Messages, "warn: "+msg) }
func (m *MockReporter) Error(msg string)   { m.Messages = append(m.Messages, "error: "+msg) }

func (m *MockReporter) Header(title string) {
	m.Messages = append(m.Messages, "header: "+title)
}

func (m *MockReporter) Box(title string, lines []string) {
	m.Messages = append(m.Messages, "box: "+title)
	m.Boxes = append(m.Boxes, BoxCall{Title: title, Lines: append([]string(nil), lines...)})
}

func (m *MockReporter) Table(headers []string, rows [][]string) {
	m.Messages = append(m.Messages, "table: "+strings.Join(headers, ","))
	clonedRows := make([][]string, len(rows))
	for i, r := range rows {
		clonedRows[i] = append([]string(nil), r...)
	}
	m.Tables = append(m.Tables, TableCall{
		Headers: append([]string(nil), headers...),
		Rows:    clonedRows,
	})
}

func (m *MockReporter) Spin(msg string) reporter.Spinner {
	idx := len(m.SpinnerCalls)
	m.SpinnerCalls = append(m.SpinnerCalls, SpinnerCall{StartMessage: msg})
	m.Messages = append(m.Messages, "spin: "+msg)
	return &mockSpinner{m: m, idx: idx}
}

func (m *MockReporter) Progress(msg string) reporter.ProgressBar {
	idx := len(m.ProgressCalls)
	m.ProgressCalls = append(m.ProgressCalls, ProgressCall{StartMessage: msg})
	m.Messages = append(m.Messages, "progress: "+msg)
	return &mockProgressBar{m: m, idx: idx}
}

func (m *MockReporter) Checklist(items []string) reporter.Checklist {
	idx := len(m.ChecklistCalls)
	m.ChecklistCalls = append(m.ChecklistCalls, ChecklistCall{
		Items: append([]string(nil), items...),
	})
	m.Messages = append(m.Messages, "checklist: "+strings.Join(items, ","))
	return &mockChecklist{m: m, idx: idx}
}

func (m *MockReporter) Data(p []byte) {
	m.Stdout = append(m.Stdout, p...)
	m.Messages = append(m.Messages, "data: "+string(p))
}

func (m *MockReporter) Diff(p []byte) {
	m.Stdout = append(m.Stdout, p...)
	m.Messages = append(m.Messages, "diff: "+string(p))
}

// HasMessage reports whether any recorded message contains the given text.
func (m *MockReporter) HasMessage(text string) bool {
	for _, msg := range m.Messages {
		if strings.Contains(msg, text) {
			return true
		}
	}
	return false
}

// Clear resets all recorded state.
func (m *MockReporter) Clear() {
	m.Messages = nil
	m.Stdout = nil
	m.Tables = nil
	m.Boxes = nil
	m.SpinnerCalls = nil
	m.ProgressCalls = nil
	m.ChecklistCalls = nil
}

type mockSpinner struct {
	m        *MockReporter
	idx      int
	finished bool
}

func (s *mockSpinner) Done(msg string) {
	if s.finished {
		return
	}
	s.finished = true
	s.m.SpinnerCalls[s.idx].Outcome = "done"
	s.m.SpinnerCalls[s.idx].EndMessage = msg
	s.m.Messages = append(s.m.Messages, "spin-done: "+msg)
}

func (s *mockSpinner) Fail(msg string) {
	if s.finished {
		return
	}
	s.finished = true
	s.m.SpinnerCalls[s.idx].Outcome = "fail"
	s.m.SpinnerCalls[s.idx].EndMessage = msg
	s.m.Messages = append(s.m.Messages, "spin-fail: "+msg)
}

func (s *mockSpinner) Stop() {
	if s.finished {
		return
	}
	s.finished = true
	s.m.SpinnerCalls[s.idx].Outcome = "stop"
	s.m.Messages = append(s.m.Messages, "spin-stop")
}

type mockProgressBar struct {
	m        *MockReporter
	idx      int
	finished bool
}

func (p *mockProgressBar) Update(percent float64, msg string) {
	if p.finished {
		return
	}
	p.m.ProgressCalls[p.idx].Updates = append(p.m.ProgressCalls[p.idx].Updates, ProgressUpdate{
		Percent: percent,
		Message: msg,
	})
}

func (p *mockProgressBar) Done(msg string) {
	if p.finished {
		return
	}
	p.finished = true
	p.m.ProgressCalls[p.idx].Outcome = "done"
	p.m.ProgressCalls[p.idx].EndMessage = msg
	p.m.Messages = append(p.m.Messages, "progress-done: "+msg)
}

func (p *mockProgressBar) Fail(msg string) {
	if p.finished {
		return
	}
	p.finished = true
	p.m.ProgressCalls[p.idx].Outcome = "fail"
	p.m.ProgressCalls[p.idx].EndMessage = msg
	p.m.Messages = append(p.m.Messages, "progress-fail: "+msg)
}

func (p *mockProgressBar) Stop() {
	if p.finished {
		return
	}
	p.finished = true
	p.m.ProgressCalls[p.idx].Outcome = "stop"
	p.m.Messages = append(p.m.Messages, "progress-stop")
}

type mockChecklist struct {
	m      *MockReporter
	idx    int
	closed bool
}

func (c *mockChecklist) Start(idx int)            { c.record(idx, "start", "") }
func (c *mockChecklist) Done(idx int, msg string) { c.record(idx, "done", msg) }
func (c *mockChecklist) Fail(idx int, msg string) { c.record(idx, "fail", msg) }
func (c *mockChecklist) Skip(idx int, msg string) { c.record(idx, "skip", msg) }

func (c *mockChecklist) Close() {
	if c.closed {
		return
	}
	c.closed = true
	c.m.ChecklistCalls[c.idx].Closed = true
	c.m.Messages = append(c.m.Messages, "checklist-close")
}

func (c *mockChecklist) record(idx int, outcome, msg string) {
	if c.closed {
		return
	}
	c.m.ChecklistCalls[c.idx].Transitions = append(c.m.ChecklistCalls[c.idx].Transitions,
		ChecklistTransition{Index: idx, Outcome: outcome, Message: msg})
	tag := "checklist-" + outcome
	if msg != "" {
		c.m.Messages = append(c.m.Messages, tag+": "+msg)
	} else {
		c.m.Messages = append(c.m.Messages, tag)
	}
}
