package example

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestOnlyCanceled(t *testing.T) {
	for _, err := range []error{context.Canceled, fmt.Errorf("read: %w", context.Canceled), errors.Join(context.Canceled, context.Canceled)} {
		if !onlyCanceled(err) {
			t.Fatal(err)
		}
	}
	for _, err := range []error{nil, context.DeadlineExceeded, errors.New("close failed"), errors.Join(context.Canceled, errors.New("close failed")), fmt.Errorf("run: %w", errors.Join(context.Canceled, errors.New("close failed")))} {
		if onlyCanceled(err) {
			t.Fatal("suppressed error", err)
		}
	}
}
