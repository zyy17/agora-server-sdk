package messaging

import (
	"context"
	"errors"
	"testing"
)

func TestHistoryValidation(t *testing.T) {
	messageChannel := Channel{Name: "room", Type: MessageChannel}
	tests := []struct {
		name      string
		channel   Channel
		options   HistoryOptions
		wantCount int
		invalid   bool
	}{
		{name: "default count", channel: messageChannel, wantCount: 100},
		{name: "user channel", channel: Channel{Name: "alice", Type: UserChannel}, wantCount: 100},
		{name: "minimum count", channel: messageChannel, options: HistoryOptions{Count: 1}, wantCount: 1},
		{name: "maximum count", channel: messageChannel, options: HistoryOptions{Count: 100}, wantCount: 100},
		{name: "start only", channel: messageChannel, options: HistoryOptions{Start: 2000}, wantCount: 100},
		{name: "end only", channel: messageChannel, options: HistoryOptions{End: 1000}, wantCount: 100},
		{name: "newest to oldest", channel: messageChannel, options: HistoryOptions{Start: 2000, End: 1000, Count: 50}, wantCount: 50},
		{name: "equal bounds", channel: messageChannel, options: HistoryOptions{Start: 1000, End: 1000}, wantCount: 100},
		{name: "stream channel", channel: Channel{Name: "room", Type: StreamChannel}, invalid: true},
		{name: "unset channel type", channel: Channel{Name: "room"}, invalid: true},
		{name: "unknown channel type", channel: Channel{Name: "room", Type: ChannelType(99)}, invalid: true},
		{name: "empty channel name", channel: Channel{Type: MessageChannel}, invalid: true},
		{name: "NUL in channel name", channel: Channel{Name: "a\x00b", Type: UserChannel}, invalid: true},
		{name: "invalid UTF-8 channel name", channel: Channel{Name: "\xff", Type: MessageChannel}, invalid: true},
		{name: "negative count", channel: messageChannel, options: HistoryOptions{Count: -1}, invalid: true},
		{name: "count exceeds service limit", channel: messageChannel, options: HistoryOptions{Count: 101}, invalid: true},
		{name: "uint16 maximum is not service limit", channel: messageChannel, options: HistoryOptions{Count: 65535}, invalid: true},
		{name: "count exceeds uint16", channel: messageChannel, options: HistoryOptions{Count: 65536}, invalid: true},
		{name: "negative start", channel: messageChannel, options: HistoryOptions{Start: -1}, invalid: true},
		{name: "negative end", channel: messageChannel, options: HistoryOptions{End: -1}, invalid: true},
		{name: "oldest to newest is invalid", channel: messageChannel, options: HistoryOptions{Start: 1000, End: 2000}, invalid: true},
	}

	c := testClient(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Valid inputs reach cancellation without invoking a native SDK method.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := historyQueryOptions(tt.channel, tt.options)
			if tt.invalid {
				if !errors.Is(err, ErrInvalidArgument) || got != nil {
					t.Fatalf("options = %+v, error = %v; want invalid argument and no options", got, err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if got.Start != tt.options.Start || got.End != tt.options.End || got.Count != tt.wantCount {
					t.Fatalf("options = %+v; want start %d, end %d, count %d", got, tt.options.Start, tt.options.End, tt.wantCount)
				}
			}

			_, err = c.History(ctx, tt.channel, tt.options)
			wantErr := context.Canceled
			if tt.invalid {
				wantErr = ErrInvalidArgument
			}
			if !errors.Is(err, wantErr) {
				t.Fatalf("History error = %v; want %v", err, wantErr)
			}
		})
	}
}

func TestHistoryDoesNotBroadenOtherChannelValidation(t *testing.T) {
	for _, channelType := range []ChannelType{MessageChannel, StreamChannel} {
		if err := (Channel{Name: "room", Type: channelType}).validate(); err != nil {
			t.Fatalf("channel type %v: %v", channelType, err)
		}
	}
	if err := (Channel{Name: "alice", Type: UserChannel}).validate(); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("other channel operations accepted a user channel: %v", err)
	}
}
