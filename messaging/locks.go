package messaging

import (
	"context"
	"math"
	"time"
)

func validateLock(ch Channel, name string) error {
	if err := ch.validate(); err != nil {
		return err
	}
	return validText("lock name", name, true)
}

// SetLock defines a lock. TTL must be a positive whole number of seconds; the
// SDK validates its server-side allowed range.
func (c *Client) SetLock(ctx context.Context, ch Channel, name string, ttl time.Duration) error {
	if err := validateLock(ch, name); err != nil {
		return err
	}
	if ttl <= 0 || ttl%time.Second != 0 || ttl/time.Second > math.MaxUint32 {
		return invalid("lock TTL must be positive whole seconds")
	}
	return c.call(ctx, "set lock", func() (int, uint64) {
		n := c.native.GetLock()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.SetLock(ch.Name, ch.Type, name, uint32(ttl/time.Second), &id)
		return code, id
	})
}
func (c *Client) Locks(ctx context.Context, ch Channel) ([]Lock, error) {
	if err := ch.validate(); err != nil {
		return nil, err
	}
	v, err := c.request(ctx, "get locks", nil, func() (int, uint64) {
		n := c.native.GetLock()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.GetLocks(ch.Name, ch.Type, &id)
		return code, id
	})
	if err != nil {
		return nil, err
	}
	return v.([]Lock), nil
}
func (c *Client) RemoveLock(ctx context.Context, ch Channel, name string) error {
	if err := validateLock(ch, name); err != nil {
		return err
	}
	return c.call(ctx, "remove lock", func() (int, uint64) {
		n := c.native.GetLock()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.RemoveLock(ch.Name, ch.Type, name, &id)
		return code, id
	})
}

// AcquireLock requests ownership. Automatic native retry is disabled, so a
// canceled wait never leaves an unbounded acquisition retry running.
func (c *Client) AcquireLock(ctx context.Context, ch Channel, name string) error {
	if err := validateLock(ch, name); err != nil {
		return err
	}
	return c.call(ctx, "acquire lock", func() (int, uint64) {
		n := c.native.GetLock()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.AcquireLock(ch.Name, ch.Type, name, false, &id)
		return code, id
	})
}
func (c *Client) ReleaseLock(ctx context.Context, ch Channel, name string) error {
	if err := validateLock(ch, name); err != nil {
		return err
	}
	return c.call(ctx, "release lock", func() (int, uint64) {
		n := c.native.GetLock()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.ReleaseLock(ch.Name, ch.Type, name, &id)
		return code, id
	})
}
func (c *Client) RevokeLock(ctx context.Context, ch Channel, name, owner string) error {
	if err := validateLock(ch, name); err != nil {
		return err
	}
	if err := validText("lock owner", owner, true); err != nil {
		return err
	}
	return c.call(ctx, "revoke lock", func() (int, uint64) {
		n := c.native.GetLock()
		if n == nil {
			return unavailableCode, 0
		}
		var id uint64
		code := n.RevokeLock(ch.Name, ch.Type, name, owner, &id)
		return code, id
	})
}
