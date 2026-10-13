package messaging

import (
	"context"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

// History returns one page of stored messages for a MessageChannel or
// UserChannel; stream-channel history is not supported. User-channel history
// addresses a user channel, not a bidirectional conversation log. Messages must
// have been published with PublishOptions.StoreInHistory, and the project must
// have history access.
//
// Count is 1 through 100, with zero selecting 100. Start is the exclusive newer
// bound, End the inclusive older bound, and zero leaves either bound unset.
// Nonzero bounds must satisfy Start >= End. All timestamps retain native SDK
// units, and native interval and retention limits still apply.
//
// To retrieve older pages, pass the previous NextStart as Start, checking that it
// fits int64 before conversion and retaining the other query options. NextStart
// of zero means there are no more pages. The returned payloads are Go-owned.
// On any error, History returns an empty page rather than partial callback data.
func (c *Client) History(ctx context.Context, ch Channel, o HistoryOptions) (HistoryPage, error) {
	options, err := historyQueryOptions(ch, o)
	if err != nil {
		return HistoryPage{}, err
	}
	v, err := c.request(ctx, "history", nil, func() (int, uint64) {
		n := c.native.GetHistory()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.GetMessages(ch.Name, ch.Type, options, &id)
		return code, id
	})
	if err != nil {
		return HistoryPage{}, err
	}
	return v.(HistoryPage), nil
}

func historyQueryOptions(ch Channel, o HistoryOptions) (*native.GetHistoryMessagesOptions, error) {
	if err := validText("channel name", ch.Name, true); err != nil {
		return nil, err
	}
	// History supports direct-user inboxes, but not stream-channel topics.
	// Other channel operations retain their own Message/Stream validation.
	if ch.Type != MessageChannel && ch.Type != UserChannel {
		return nil, invalid("history channel type")
	}
	if o.Count < 0 || o.Count > 100 {
		return nil, invalid("history count must be 0..100")
	}
	if o.Start < 0 || o.End < 0 {
		return nil, invalid("negative history timestamp")
	}
	if o.Start != 0 && o.End != 0 && o.Start < o.End {
		return nil, invalid("history start precedes end")
	}
	if o.Count == 0 {
		o.Count = 100
	}
	return &native.GetHistoryMessagesOptions{Start: o.Start, End: o.End, Count: o.Count}, nil
}
