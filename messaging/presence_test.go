package messaging

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

func TestStateItemsSortedAndCopied(t *testing.T) {
	state := map[string]string{"zebra": "last", "alpha": "first", "empty": ""}
	items, err := stateItems(state)
	if err != nil {
		t.Fatal(err)
	}
	state["alpha"] = "changed"
	want := []*native.StateItem{{Key: "alpha", Value: "first"}, {Key: "empty", Value: ""}, {Key: "zebra", Value: "last"}}
	if !reflect.DeepEqual(items, want) {
		t.Fatalf("got %+v, want %+v", items, want)
	}
}

func TestStateItemsValidation(t *testing.T) {
	for _, state := range []map[string]string{nil, {}, {"": "value"}, {"bad\x00key": "value"}, {"key": "bad\x00value"}} {
		if items, err := stateItems(state); !errors.Is(err, ErrInvalidArgument) || items != nil {
			t.Fatalf("state %v: got %v, %v", state, items, err)
		}
	}
}

func TestSetStateValidationOrder(t *testing.T) {
	client := testClient(t, 1)
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if err := client.SetState(context.Background(), Channel{}, nil); !errors.Is(err, ErrInvalidArgument) || !strings.Contains(err.Error(), "channel name") {
		t.Fatalf("channel validation must precede state validation: %v", err)
	}
	ch := Channel{Name: "room", Type: MessageChannel}
	if err := client.SetState(context.Background(), ch, nil); !errors.Is(err, ErrInvalidArgument) || !strings.Contains(err.Error(), "empty state") {
		t.Fatalf("state validation must precede closed check: %v", err)
	}
	if err := client.SetState(context.Background(), ch, map[string]string{"key": "value"}); !errors.Is(err, ErrClosed) {
		t.Fatalf("got %v, want closed", err)
	}
}
