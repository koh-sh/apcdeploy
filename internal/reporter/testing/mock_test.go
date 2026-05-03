package testing

import (
	"errors"
	"reflect"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

func TestMockReporter_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ reporter.Reporter = (*MockReporter)(nil)
}

func TestMockReporter_RecordsAllKinds(t *testing.T) {
	t.Parallel()

	m := &MockReporter{}
	m.Step("s1")
	m.Success("s2")
	m.Info("s3")
	m.Warn("s4")
	m.Error("s5")
	m.Header("h1")
	m.Box("b1", []string{"l1", "l2"})
	m.Table([]string{"col"}, [][]string{{"v"}})
	m.Data([]byte("d1"))
	m.Diff([]byte("+x\n"))

	wantPrefixes := []string{
		"step: s1",
		"success: s2",
		"info: s3",
		"warn: s4",
		"error: s5",
		"header: h1",
		"box: b1",
		"table: col",
		"data: d1",
		"diff: +x\n",
	}
	for _, want := range wantPrefixes {
		if !m.HasMessage(want) {
			t.Errorf("expected message containing %q; got %v", want, m.Messages)
		}
	}

	if string(m.Stdout) != "d1+x\n" {
		t.Errorf("Stdout = %q, want %q", string(m.Stdout), "d1+x\n")
	}

	if len(m.Tables) != 1 || !reflect.DeepEqual(m.Tables[0].Headers, []string{"col"}) {
		t.Errorf("expected one table with headers [col]; got %+v", m.Tables)
	}
	if len(m.Boxes) != 1 || !reflect.DeepEqual(m.Boxes[0].Lines, []string{"l1", "l2"}) {
		t.Errorf("expected one box with two lines; got %+v", m.Boxes)
	}
}

func TestMockReporter_SpinnerLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("done outcome", func(t *testing.T) {
		t.Parallel()
		m := &MockReporter{}
		sp := m.Spin("starting")
		sp.Done("finished")

		if len(m.SpinnerCalls) != 1 {
			t.Fatalf("expected 1 spinner call; got %d", len(m.SpinnerCalls))
		}
		got := m.SpinnerCalls[0]
		if got.StartMessage != "starting" || got.Outcome != "done" || got.EndMessage != "finished" {
			t.Errorf("unexpected spinner call: %+v", got)
		}

		// Second Done is a no-op
		sp.Done("again")
		if m.SpinnerCalls[0].EndMessage != "finished" {
			t.Errorf("second Done() should be ignored; got %+v", m.SpinnerCalls[0])
		}
	})

	t.Run("fail outcome", func(t *testing.T) {
		t.Parallel()
		m := &MockReporter{}
		sp := m.Spin("starting")
		sp.Fail("crashed")

		got := m.SpinnerCalls[0]
		if got.Outcome != "fail" || got.EndMessage != "crashed" {
			t.Errorf("unexpected spinner call: %+v", got)
		}
	})
}

func TestMockReporter_TargetsLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("records ids and transitions in order", func(t *testing.T) {
		t.Parallel()
		m := &MockReporter{}
		tg := m.Targets([]string{"a", "b"})
		tg.SetPhase("a", "preparing", "")
		tg.SetProgress("a", 0.5, 0)
		tg.Done("a", "deployed")
		tg.Fail("b", errors.New("boom"))
		tg.Close()

		if len(m.TargetsCalls) != 1 {
			t.Fatalf("expected 1 Targets call; got %d", len(m.TargetsCalls))
		}
		got := m.TargetsCalls[0]
		if !got.Closed {
			t.Error("expected Closed=true after Close()")
		}
		if !reflect.DeepEqual(got.IDs, []string{"a", "b"}) {
			t.Errorf("IDs = %v, want [a b]", got.IDs)
		}
		wantKinds := []string{"phase", "progress", "done", "fail"}
		gotKinds := make([]string, 0, len(got.Transitions))
		for _, tr := range got.Transitions {
			gotKinds = append(gotKinds, tr.Kind)
		}
		if !reflect.DeepEqual(gotKinds, wantKinds) {
			t.Errorf("transition order = %v, want %v", gotKinds, wantKinds)
		}
	})

	t.Run("double Close is no-op", func(t *testing.T) {
		t.Parallel()
		m := &MockReporter{}
		tg := m.Targets([]string{"a"})
		tg.Close()
		before := len(m.Messages)
		tg.Close()
		if len(m.Messages) != before {
			t.Errorf("second Close must not append a message; got %v", m.Messages)
		}
	})

	t.Run("transitions after Close are silent", func(t *testing.T) {
		t.Parallel()
		m := &MockReporter{}
		tg := m.Targets([]string{"a"})
		tg.Close()
		tg.SetPhase("a", "preparing", "")
		tg.Done("a", "x")
		tg.Fail("a", errors.New("y"))
		tg.Skip("a", "z")
		if got := len(m.TargetsCalls[0].Transitions); got != 0 {
			t.Errorf("post-Close transitions must be ignored; got %d", got)
		}
	})

	t.Run("Clear resets TargetsCalls", func(t *testing.T) {
		t.Parallel()
		m := &MockReporter{}
		tg := m.Targets([]string{"a"})
		tg.Done("a", "x")
		tg.Close()
		m.Clear()
		if len(m.TargetsCalls) != 0 {
			t.Errorf("Clear must zero TargetsCalls; got %+v", m.TargetsCalls)
		}
	})
}

func TestMockReporter_HasMessage(t *testing.T) {
	t.Parallel()

	m := &MockReporter{}
	if m.HasMessage("anything") {
		t.Error("empty MockReporter should have no messages")
	}
	m.Step("hello world")
	if !m.HasMessage("hello") {
		t.Error("HasMessage should match a substring")
	}
	if m.HasMessage("missing") {
		t.Error("HasMessage should not match unrelated text")
	}
}

func TestMockReporter_Clear(t *testing.T) {
	t.Parallel()

	m := &MockReporter{}
	m.Step("x")
	m.Data([]byte("y"))
	m.Table([]string{"a"}, [][]string{{"b"}})
	m.Box("t", []string{"l"})
	m.Spin("z").Done("done")

	m.Clear()

	if len(m.Messages) != 0 || len(m.Stdout) != 0 || len(m.Tables) != 0 || len(m.Boxes) != 0 || len(m.SpinnerCalls) != 0 {
		t.Errorf("Clear should reset all state; got %+v", m)
	}
}
