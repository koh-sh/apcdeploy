package batch

import (
	"sync"
	"testing"
)

func TestPayloadCollector_PreservesArgumentOrder(t *testing.T) {
	t.Parallel()

	targets := []*Target{
		{Identifier: "us-east-1/app/p/dev"},
		{Identifier: "us-east-1/app/p/stg"},
		{Identifier: "us-east-1/app/p/prod"},
	}
	pc := NewPayloadCollector(targets)

	// Set out of argument order to confirm the collector still places
	// payloads in the same order as the original Targets slice.
	pc.Set("us-east-1/app/p/prod", []byte("prod-body"), true)
	pc.Set("us-east-1/app/p/dev", []byte("dev-body"), true)
	pc.Set("us-east-1/app/p/stg", nil, false)

	payloads := pc.Payloads()
	wantPayloads := []string{"dev-body", "", "prod-body"}
	for i, want := range wantPayloads {
		if string(payloads[i]) != want {
			t.Errorf("payloads[%d] = %q, want %q", i, payloads[i], want)
		}
	}

	hasChanges := pc.HasChanges()
	wantChanges := []bool{true, false, true}
	for i, want := range wantChanges {
		if hasChanges[i] != want {
			t.Errorf("hasChanges[%d] = %v, want %v", i, hasChanges[i], want)
		}
	}
}

func TestPayloadCollector_IgnoresUnknownIdentifier(t *testing.T) {
	t.Parallel()

	pc := NewPayloadCollector([]*Target{
		{Identifier: "known"},
	})
	// Setting an unknown identifier must NOT panic, allocate, or mutate
	// the slot for "known".
	pc.Set("unknown", []byte("ignored"), true)

	if pc.Payloads()[0] != nil {
		t.Errorf("payload for 'known' = %q, want nil (only 'unknown' was Set)", pc.Payloads()[0])
	}
	if pc.HasChanges()[0] != false {
		t.Errorf("hasChanges for 'known' = true, want false")
	}
}

func TestPayloadCollector_IsGoroutineSafe(t *testing.T) {
	t.Parallel()

	const n = 50
	targets := make([]*Target, n)
	for i := range n {
		targets[i] = &Target{Identifier: itoa(i)}
	}
	pc := NewPayloadCollector(targets)

	// Concurrent Sets from many goroutines must not race; -race in CI
	// is the real assertion. The sequential check below guards against
	// silent slot-collision bugs.
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			pc.Set(itoa(i), []byte("v"+itoa(i)), i%2 == 0)
		}(i)
	}
	wg.Wait()

	for i := range n {
		want := "v" + itoa(i)
		if string(pc.Payloads()[i]) != want {
			t.Errorf("slot %d = %q, want %q", i, pc.Payloads()[i], want)
		}
		if pc.HasChanges()[i] != (i%2 == 0) {
			t.Errorf("hasChanges[%d] = %v, want %v", i, pc.HasChanges()[i], i%2 == 0)
		}
	}
}

// itoa is a tiny strconv.Itoa replacement so this file does not pull in
// strconv just for test fixture identifiers.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
