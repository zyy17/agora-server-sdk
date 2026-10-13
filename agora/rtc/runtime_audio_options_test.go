package agoraservice

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestRuntimeBooleanOptionRetriesFailuresAndRejectsConflicts(t *testing.T) {
	var option runtimeOption[bool]
	calls := 0
	if code := option.apply(false, func(bool) int { calls++; return -17 }); code != -17 || option.configured {
		t.Fatal("failed native setter committed state")
	}
	if code := option.apply(false, func(value bool) int {
		calls++
		if value {
			t.Fatal(value)
		}
		return 0
	}); code != 0 || !option.configured {
		t.Fatal("retry failed")
	}
	if code := option.apply(false, func(bool) int { t.Fatal("identical request called native setter"); return 0 }); code != 0 {
		t.Fatal(code)
	}
	if code := option.apply(true, func(bool) int { t.Fatal("conflict changed another connection's option"); return 0 }); code != -2002 || option.value {
		t.Fatal("conflicting request accepted")
	}
	if calls != 2 {
		t.Fatal(calls)
	}
	option.reset()
	if code := option.apply(true, func(value bool) int {
		if !value {
			t.Fatal(value)
		}
		calls++
		return 0
	}); code != 0 || !option.value || calls != 3 {
		t.Fatal("new runtime retained old setting")
	}
}

func TestRuntimeBooleanOptionSerializesFirstRequest(t *testing.T) {
	var option runtimeOption[bool]
	var calls atomic.Int32
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if code := option.apply(i%2 == 0, func(bool) int { calls.Add(1); return 0 }); code != 0 && code != -2002 {
				t.Error(code)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatal("raced engine-wide updates:", calls.Load())
	}
}
