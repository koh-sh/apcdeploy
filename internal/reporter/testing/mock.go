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

	// TargetsCalls records each Targets lifecycle: the initial identifier
	// list and every recorded transition.
	TargetsCalls []TargetsCall
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
// to Spin, the sequence of Update labels mid-flight, plus the terminating
// Done/Fail outcome and message.
type SpinnerCall struct {
	StartMessage string
	Updates      []string
	Outcome      string // "done", "fail", or "stop"
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
	m.TargetsCalls = nil
}

type mockSpinner struct {
	m        *MockReporter
	idx      int
	finished bool
}

func (s *mockSpinner) Update(msg string) {
	if s.finished {
		return
	}
	s.m.SpinnerCalls[s.idx].Updates = append(s.m.SpinnerCalls[s.idx].Updates, msg)
	s.m.Messages = append(s.m.Messages, "spin-update: "+msg)
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
