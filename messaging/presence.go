package messaging

import (
	"context"
	"sort"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

// OnlineUsers returns one page of users in a message or stream channel.
// User IDs are requested unless o.ExcludeUserID is true; o.IncludeState requests state.
// Use an empty o.Page initially, then pass each nonempty NextPage unchanged.
// Membership can change between pages, so the result is not an atomic snapshot.
// The method does not subscribe to presence events.
//
// The returned data is Go-owned. On an SDK or context error, the returned page
// is empty, even if a native callback also supplied partial data.
func (c *Client) OnlineUsers(ctx context.Context, ch Channel, o PresenceOptions) (PresencePage, error) {
	if err := ch.validate(); err != nil {
		return PresencePage{}, err
	}
	if err := validText("presence page", o.Page, false); err != nil {
		return PresencePage{}, err
	}
	options := o.nativeOptions()
	v, err := c.request(ctx, "online users", nil, func() (int, uint64) {
		n := c.native.GetPresence()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.GetOnlineUsers(ch.Name, ch.Type, options, &id)
		return code, id
	})
	if err != nil {
		return PresencePage{}, err
	}
	return v.(PresencePage), nil
}

// UserChannels queries the message and stream channels associated with userID.
// The user ID must be nonempty UTF-8 without NUL. It returns a Go-owned list
// reported by the SDK, or nil on error; it does not create any subscription.
func (c *Client) UserChannels(ctx context.Context, userID string) ([]Channel, error) {
	if err := validText("user ID", userID, true); err != nil {
		return nil, err
	}
	v, err := c.request(ctx, "user channels", nil, func() (int, uint64) {
		n := c.native.GetPresence()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.GetUserChannels(userID, &id)
		return code, id
	})
	if err != nil {
		return nil, err
	}
	return v.([]Channel), nil
}

// SetState sets the current client's channel-scoped presence keys in a message
// or stream channel. State must be nonempty, with nonempty keys; keys and values
// must be UTF-8 without NUL. Existing keys not supplied are not requested for
// removal. Use RemoveState to remove keys.
//
// The map is copied before submission; do not mutate it concurrently with the
// call. Presence state is not durable metadata. Cancellation does not undo an
// already submitted update.
func (c *Client) SetState(ctx context.Context, ch Channel, state map[string]string) error {
	if err := ch.validate(); err != nil {
		return err
	}
	items, err := stateItems(state)
	if err != nil {
		return err
	}
	return c.call(ctx, "set state", func() (int, uint64) {
		n := c.native.GetPresence()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.SetState(ch.Name, ch.Type, items, uint(len(items)), &id)
		return code, id
	})
}

func stateItems(state map[string]string) ([]*native.StateItem, error) {
	if len(state) == 0 {
		return nil, invalid("empty state")
	}
	keys := make([]string, 0, len(state))
	for k, v := range state {
		if err := validText("state key", k, true); err != nil {
			return nil, err
		}
		if err := validText("state value", v, false); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	items := make([]*native.StateItem, 0, len(keys))
	for _, k := range keys {
		items = append(items, &native.StateItem{Key: k, Value: state[k]})
	}
	return items, nil
}

// RemoveState removes the supplied presence keys for the current user in a
// message or stream channel. An empty key list removes all state. Supplied keys
// must be nonempty UTF-8 without NUL. The key list is copied before submission.
// Cancellation does not undo an already submitted removal.
func (c *Client) RemoveState(ctx context.Context, ch Channel, keys ...string) error {
	if err := ch.validate(); err != nil {
		return err
	}
	for _, k := range keys {
		if err := validText("state key", k, true); err != nil {
			return err
		}
	}
	copied := append([]string(nil), keys...)
	return c.call(ctx, "remove state", func() (int, uint64) {
		n := c.native.GetPresence()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.RemoveState(ch.Name, ch.Type, copied, uint(len(copied)), &id)
		return code, id
	})
}

// State queries userID's presence state in a message or stream channel.
// The user ID must be nonempty UTF-8 without NUL. The returned map is Go-owned;
// on any error, the returned UserState is its zero value.
func (c *Client) State(ctx context.Context, ch Channel, userID string) (UserState, error) {
	if err := ch.validate(); err != nil {
		return UserState{}, err
	}
	if err := validText("user ID", userID, true); err != nil {
		return UserState{}, err
	}
	v, err := c.request(ctx, "get state", nil, func() (int, uint64) {
		n := c.native.GetPresence()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.GetState(ch.Name, ch.Type, userID, &id)
		return code, id
	})
	if err != nil {
		return UserState{}, err
	}
	return v.(UserState), nil
}

// This private code never collides with the vendor's current error range and
// is translated to the public sentinel rather than exposed as an SDK rejection.
const unavailableCode = -2147483647
