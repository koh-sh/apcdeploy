package testing

import (
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

func TestMockReporter_ProgressLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("done outcome with updates", func(t *testing.T) {
		t.Parallel()
		m := &MockReporter{}
		pb := m.Progress("starting")
		pb.Update(25, "quarter")
		pb.Update(75, "three quarters")
		pb.Done("finished")

		if len(m.ProgressCalls) != 1 {
			t.Fatalf("expected 1 progress call; got %d", len(m.ProgressCalls))
		}
		got := m.ProgressCalls[0]
		if got.StartMessage != "starting" || got.Outcome != "done" || got.EndMessage != "finished" {
			t.Errorf("unexpected progress call: %+v", got)
		}
		if len(got.Updates) != 2 {
			t.Errorf("expected 2 updates; got %d", len(got.Updates))
		}
		if got.Updates[1].Percent != 75 || got.Updates[1].Message != "three quarters" {
			t.Errorf("unexpected second update: %+v", got.Updates[1])
		}

		// Second Done is a no-op; Update after finish is also a no-op.
		pb.Done("again")
		pb.Update(100, "after-done")
		if m.ProgressCalls[0].EndMessage != "finished" || len(m.ProgressCalls[0].Updates) != 2 {
			t.Errorf("post-finish calls should be ignored; got %+v", m.ProgressCalls[0])
		}
	})

	t.Run("fail outcome", func(t *testing.T) {
		t.Parallel()
		m := &MockReporter{}
		pb := m.Progress("starting")
		pb.Fail("crashed")

		got := m.ProgressCalls[0]
		if got.Outcome != "fail" || got.EndMessage != "crashed" {
			t.Errorf("unexpected progress call: %+v", got)
		}

		// Second Fail is a no-op.
		pb.Fail("again")
		if m.ProgressCalls[0].EndMessage != "crashed" {
			t.Errorf("second Fail should be ignored; got %+v", m.ProgressCalls[0])
		}
	})

	t.Run("stop outcome", func(t *testing.T) {
		t.Parallel()
		m := &MockReporter{}
		pb := m.Progress("starting")
		pb.Stop()

		got := m.ProgressCalls[0]
		if got.Outcome != "stop" || got.EndMessage != "" {
			t.Errorf("Stop should record outcome=stop with empty EndMessage; got %+v", got)
		}

		// Subsequent terminators must be ignored.
		pb.Done("late done")
		pb.Fail("late fail")
		pb.Stop()
		if m.ProgressCalls[0].Outcome != "stop" || m.ProgressCalls[0].EndMessage != "" {
			t.Errorf("post-Stop terminators must be no-ops; got %+v", m.ProgressCalls[0])
		}
	})
}

func TestMockReporter_ChecklistLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("records all transitions and Close", func(t *testing.T) {
		t.Parallel()
		m := &MockReporter{}
		chk := m.Checklist([]string{"alpha", "beta", "gamma"})

		chk.Start(0)
		chk.Done(0, "alpha-done")
		chk.Skip(1, "beta-skipped")
		chk.Fail(2, "gamma-failed")
		chk.Close()

		if len(m.ChecklistCalls) != 1 {
			t.Fatalf("expected 1 checklist call; got %d", len(m.ChecklistCalls))
		}
		got := m.ChecklistCalls[0]
		if !got.Closed {
			t.Error("expected Closed=true after Close()")
		}
		wantLabels := []string{"alpha", "beta", "gamma"}
		for i, l := range wantLabels {
			if got.Items[i] != l {
				t.Errorf("Items[%d] = %q, want %q", i, got.Items[i], l)
			}
		}
		// Start, Done(0), Skip(1), Fail(2) — 4 transitions in order.
		if len(got.Transitions) != 4 {
			t.Fatalf("expected 4 transitions; got %d (%+v)", len(got.Transitions), got.Transitions)
		}
		want := []ChecklistTransition{
			{Index: 0, Outcome: "start", Message: ""},
			{Index: 0, Outcome: "done", Message: "alpha-done"},
			{Index: 1, Outcome: "skip", Message: "beta-skipped"},
			{Index: 2, Outcome: "fail", Message: "gamma-failed"},
		}
		for i, w := range want {
			if got.Transitions[i] != w {
				t.Errorf("Transitions[%d] = %+v, want %+v", i, got.Transitions[i], w)
			}
		}
	})

	t.Run("double Close is no-op", func(t *testing.T) {
		t.Parallel()
		m := &MockReporter{}
		chk := m.Checklist([]string{"x"})
		chk.Close()
		closeMessages := len(m.Messages)
		chk.Close()
		if len(m.Messages) != closeMessages {
			t.Errorf("second Close must not append a message; got %v", m.Messages)
		}
	})

	t.Run("transitions after Close are silent", func(t *testing.T) {
		t.Parallel()
		m := &MockReporter{}
		chk := m.Checklist([]string{"x"})
		chk.Close()
		chk.Start(0)
		chk.Done(0, "ignored")
		chk.Fail(0, "ignored")
		chk.Skip(0, "ignored")
		// Only "checklist:" + "checklist-close" should be present.
		if len(m.ChecklistCalls[0].Transitions) != 0 {
			t.Errorf("post-Close transitions must be ignored; got %+v", m.ChecklistCalls[0].Transitions)
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
	m.Progress("p").Done("p-done")

	m.Clear()

	if len(m.Messages) != 0 || len(m.Stdout) != 0 || len(m.Tables) != 0 || len(m.Boxes) != 0 || len(m.SpinnerCalls) != 0 || len(m.ProgressCalls) != 0 {
		t.Errorf("Clear should reset all state; got %+v", m)
	}
}
