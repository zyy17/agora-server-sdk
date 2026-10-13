package messaging

import (
	"context"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

// History returns one page. Start and End use the native SDK's timestamp unit;
// pass the previous result's NextStart for pagination. Count defaults to 100.
func (c *Client) History(ctx context.Context, ch Channel, o HistoryOptions) (HistoryPage, error) {
	if err := ch.validate(); err != nil {
		return HistoryPage{}, err
	}
	if o.Count < 0 || o.Count > 65535 {
		return HistoryPage{}, invalid("history count must be 0..65535")
	}
	if o.Start < 0 || o.End < 0 {
		return HistoryPage{}, invalid("negative history timestamp")
	}
	if o.Start != 0 && o.End != 0 && o.Start > o.End {
		return HistoryPage{}, invalid("history start exceeds end")
	}
	if o.Count == 0 {
		o.Count = 100
	}
	options := &native.GetHistoryMessagesOptions{Start: o.Start, End: o.End, Count: o.Count}
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
