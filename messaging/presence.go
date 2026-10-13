package messaging

import (
	"context"
	"sort"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

func (c *Client) OnlineUsers(ctx context.Context, ch Channel, o PresenceOptions) (PresencePage, error) {
	if err := ch.validate(); err != nil {
		return PresencePage{}, err
	}
	if err := validText("presence page", o.Page, false); err != nil {
		return PresencePage{}, err
	}
	options := &native.GetOnlineUsersOptions{IncludeUserId: true, IncludeState: o.IncludeState, Page: o.Page}
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

// RemoveState removes the supplied keys; an empty key list removes all state.
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
