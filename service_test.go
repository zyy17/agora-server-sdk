package agorasdk

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestServiceParametersBeforeJoin(t *testing.T) {
	var calls atomic.Int32
	service := &Service{setParameters: func(value string) int {
		calls.Add(1)
		if value != `{"example.option":true}` {
			t.Fatal(value)
		}
		return -7
	}}
	for _, value := range []string{"", "null", "[]", "invalid"} {
		if err := service.SetParameters(value); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("forwarded invalid JSON")
	}
	var native *Error
	if err := service.SetParameters(`{"example.option":true}`); !errors.As(err, &native) || native.Code != -7 {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("missing native submission")
	}
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	if err := service.SetParameters(`{"example.option":true}`); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("called native after close")
	}
}

func TestServiceParametersSerializeWithClose(t *testing.T) {
	started, finish := make(chan struct{}), make(chan struct{})
	service := &Service{setParameters: func(string) int { close(started); <-finish; return 0 }, release: func() error { return nil }}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := service.SetParameters(`{}`); err != nil {
			t.Error(err)
		}
	}()
	<-started
	go func() {
		defer wg.Done()
		if err := service.Close(); err != nil {
			t.Error(err)
		}
	}()
	close(finish)
	wg.Wait()
	if err := service.SetParameters(`{}`); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
