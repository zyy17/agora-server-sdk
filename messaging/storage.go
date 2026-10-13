package messaging

import (
	"context"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

func metadataInput(m Metadata) (*native.IMetadata, error) {
	if len(m.Items) == 0 {
		return nil, invalid("empty metadata")
	}
	out := &native.IMetadata{}
	out.SetMajorRevision(m.Revision)
	seen := make(map[string]bool, len(m.Items))
	for _, item := range m.Items {
		if err := validText("metadata key", item.Key, true); err != nil {
			return nil, err
		}
		if seen[item.Key] {
			return nil, invalid("duplicate metadata key")
		}
		seen[item.Key] = true
		if err := validText("metadata value", item.Value, false); err != nil {
			return nil, err
		}
		if err := validText("metadata author", item.AuthorUserID, false); err != nil {
			return nil, err
		}
		out.SetMetadataItem(&native.MetadataItem{Key: item.Key, Value: item.Value, AuthorUserId: item.AuthorUserID, Revision: uint64(item.Revision), UpdateTs: uint64(item.UpdatedAt)})
	}
	return out, nil
}

func metadataOptions(o MetadataOptions) (*native.MetadataOptions, error) {
	if err := validText("metadata lock name", o.LockName, false); err != nil {
		return nil, err
	}
	return &native.MetadataOptions{RecordTs: o.RecordTimestamp, RecordUserId: o.RecordUserID}, nil
}

// SetChannelMetadata creates or replaces the supplied metadata items in a
// message or stream channel. It waits for the SDK result and honors the supplied
// collection and item revisions. For unconditional writes, explicitly set both
// Metadata.Revision and every MetadataItem.Revision to -1.
//
// Items must be nonempty with unique, nonempty keys; all text must be UTF-8
// without NUL. o.LockName optionally protects the write. The service validates
// revisions, quotas, and lock ownership. Cancellation does not undo the write.
func (c *Client) SetChannelMetadata(ctx context.Context, ch Channel, m Metadata, o MetadataOptions) error {
	return c.channelMetadata(ctx, "set", ch, m, o)
}

// UpdateChannelMetadata updates existing metadata items in a message or stream
// channel and waits for the SDK result. Input validation, revision preconditions,
// optional locking, and cancellation follow SetChannelMetadata. The native SDK
// reports missing-item and revision conflicts.
func (c *Client) UpdateChannelMetadata(ctx context.Context, ch Channel, m Metadata, o MetadataOptions) error {
	return c.channelMetadata(ctx, "update", ch, m, o)
}

// RemoveChannelMetadata removes the supplied metadata keys in a message or
// stream channel and waits for the SDK result. It requires at least one item;
// an empty collection does not mean remove all. Validation and revision checks
// follow SetChannelMetadata, including the optional channel lock.
func (c *Client) RemoveChannelMetadata(ctx context.Context, ch Channel, m Metadata, o MetadataOptions) error {
	return c.channelMetadata(ctx, "remove", ch, m, o)
}

func (c *Client) channelMetadata(ctx context.Context, action string, ch Channel, m Metadata, o MetadataOptions) error {
	if err := ch.validate(); err != nil {
		return err
	}
	data, err := metadataInput(m)
	if err != nil {
		return err
	}
	options, err := metadataOptions(o)
	if err != nil {
		return err
	}
	return c.call(ctx, action+" channel metadata", func() (int, uint64) {
		n := c.native.GetStorage()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		var code int
		switch action {
		case "set":
			code = n.SetChannelMetadata(ch.Name, ch.Type, data, options, o.LockName, &id)
		case "update":
			code = n.UpdateChannelMetadata(ch.Name, ch.Type, data, options, o.LockName, &id)
		case "remove":
			code = n.RemoveChannelMetadata(ch.Name, ch.Type, data, options, o.LockName, &id)
		}
		return code, id
	})
}

// ChannelMetadata queries metadata for a message or stream channel.
// The returned collection is a Go-owned copy. On any error it returns the zero
// Metadata value; querying does not subscribe to metadata changes.
func (c *Client) ChannelMetadata(ctx context.Context, ch Channel) (Metadata, error) {
	if err := ch.validate(); err != nil {
		return Metadata{}, err
	}
	v, err := c.request(ctx, "get channel metadata", nil, func() (int, uint64) {
		n := c.native.GetStorage()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.GetChannelMetadata(ch.Name, ch.Type, &id)
		return code, id
	})
	if err != nil {
		return Metadata{}, err
	}
	return v.(Metadata), nil
}

// SetUserMetadata creates or replaces the supplied user's metadata items and
// waits for the SDK result. userID must be nonempty UTF-8 without NUL. Item
// validation and revision preconditions follow SetChannelMetadata, but a nonempty
// o.LockName is invalid for user metadata. Cancellation does not undo the write.
func (c *Client) SetUserMetadata(ctx context.Context, userID string, m Metadata, o MetadataOptions) error {
	return c.userMetadata(ctx, "set", userID, m, o)
}

// UpdateUserMetadata updates existing metadata items for userID and waits for
// the SDK result. Validation and revision checks follow SetUserMetadata. The
// native SDK reports missing-item, permission, and revision conflicts.
func (c *Client) UpdateUserMetadata(ctx context.Context, userID string, m Metadata, o MetadataOptions) error {
	return c.userMetadata(ctx, "update", userID, m, o)
}

// RemoveUserMetadata removes the supplied metadata keys for userID and waits
// for the SDK result. At least one item is required; an empty collection does not
// remove all metadata. Validation and revision checks follow SetUserMetadata.
func (c *Client) RemoveUserMetadata(ctx context.Context, userID string, m Metadata, o MetadataOptions) error {
	return c.userMetadata(ctx, "remove", userID, m, o)
}

func (c *Client) userMetadata(ctx context.Context, action, userID string, m Metadata, o MetadataOptions) error {
	if err := validText("user ID", userID, true); err != nil {
		return err
	}
	data, err := metadataInput(m)
	if err != nil {
		return err
	}
	options, err := metadataOptions(o)
	if err != nil {
		return err
	}
	if o.LockName != "" {
		return invalid("lock name only applies to channel metadata")
	}
	return c.call(ctx, action+" user metadata", func() (int, uint64) {
		n := c.native.GetStorage()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		var code int
		switch action {
		case "set":
			code = n.SetUserMetadata(userID, data, options, &id)
		case "update":
			code = n.UpdateUserMetadata(userID, data, options, &id)
		case "remove":
			code = n.RemoveUserMetadata(userID, data, options, &id)
		}
		return code, id
	})
}

// UserMetadata queries userID's metadata without subscribing to changes.
// The user ID must be nonempty UTF-8 without NUL. The result is a Go-owned copy;
// on any error it returns the zero Metadata value.
func (c *Client) UserMetadata(ctx context.Context, userID string) (Metadata, error) {
	if err := validText("user ID", userID, true); err != nil {
		return Metadata{}, err
	}
	v, err := c.request(ctx, "get user metadata", nil, func() (int, uint64) {
		n := c.native.GetStorage()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.GetUserMetadata(userID, &id)
		return code, id
	})
	if err != nil {
		return Metadata{}, err
	}
	return v.(Metadata), nil
}

// SubscribeUserMetadata requests metadata-change notifications for userID and
// waits for the SDK result. The user ID must be nonempty UTF-8 without NUL.
// Notifications arrive as EventStorage events and are subject to queue overflow.
func (c *Client) SubscribeUserMetadata(ctx context.Context, userID string) error {
	if err := validText("user ID", userID, true); err != nil {
		return err
	}
	return c.call(ctx, "subscribe user metadata", func() (int, uint64) {
		n := c.native.GetStorage()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.SubscribeUserMetadata(userID, &id)
		return code, id
	})
}

// UnsubscribeUserMetadata removes the metadata-change subscription for userID
// and waits for the SDK result. The user ID must be nonempty UTF-8 without NUL.
// Already queued storage events are not removed.
func (c *Client) UnsubscribeUserMetadata(ctx context.Context, userID string) error {
	if err := validText("user ID", userID, true); err != nil {
		return err
	}
	return c.call(ctx, "unsubscribe user metadata", func() (int, uint64) {
		n := c.native.GetStorage()
		if n == nil {
			return unavailableCode, 0
		}
		return n.UnsubscribeUserMetadata(userID)
	})
}
