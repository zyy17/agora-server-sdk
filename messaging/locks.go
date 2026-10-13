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

// SetLock defines a named lock in a message or stream channel without acquiring
// it. TTL must be a positive whole number of seconds fitting uint32; the SDK
// validates its service-specific range. TTL controls retained ownership after
// its owner disconnects, not the lifetime of the lock definition.
// Use AcquireLock to request ownership.
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

// Locks queries the defined locks and their reported owners in a message or
// stream channel. It returns a Go-owned slice, or nil on error. The result is a
// point-in-time query and does not establish ongoing ownership.
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

// RemoveLock requests deletion of a lock definition in a message or stream
// channel and waits for the SDK result. The name must be nonempty UTF-8 without
// NUL. This is distinct from ReleaseLock, which relinquishes ownership.
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

// AcquireLock requests ownership of a lock in a message or stream channel and
// waits for the SDK result. Automatic native acquisition retry is disabled.
// A timeout or cancellation does not prove acquisition failed: an already
// submitted request can still succeed. Query ownership before blindly retrying
// or performing a consequential write.
func (c *Client) AcquireLock(ctx context.Context, ch Channel, name string) error {
	return c.AcquireLockWithOptions(ctx, ch, name, AcquireLockOptions{})
}

// AcquireLockOptions controls native lock acquisition. The zero value does not retry.
type AcquireLockOptions struct {
	// Retry asks the native SDK to retry acquisition. Cancellation and the Go
	// request timeout stop waiting ONLY; they do not cancel native retry. A lock
	// can be acquired later, even after a failed acquisition callback. Enable
	// Config.ResultEvents before creating the
	// client, handle RequestAcquireLock results and EventLock notifications,
	// query ownership, and release unwanted acquisitions with a fresh context.
	// Events can overflow; do not use the event queue alone as proof of ownership.
	Retry bool
}

// AcquireLockWithOptions requests lock ownership with explicit native retry
// control. With Retry true, the SDK can acquire ownership after this method has
// returned a context error or an SDK acquisition failure. A failure callback
// is not proof that retry has stopped. The SDK exposes no per-request
// cancellation API.
// Query ownership before retrying, and release unwanted late acquisitions. See
// AcquireLockOptions.Retry and Config.ResultEvents for observing late results.
func (c *Client) AcquireLockWithOptions(ctx context.Context, ch Channel, name string, options AcquireLockOptions) error {
	if err := validateLock(ch, name); err != nil {
		return err
	}
	return c.call(ctx, "acquire lock", func() (int, uint64) {
		n := c.native.GetLock()
		if n == nil {
			return unavailableCode, 0
		}
		return submitLockAcquisition(n, ch, name, options)
	})
}

// ReleaseLock relinquishes the current user's ownership of a lock in a message
// or stream channel and waits for the SDK result. It does not remove the lock
// definition. Cancellation does not undo a submitted release.
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

// RevokeLock requests revocation of the named owner's lock in a message or
// stream channel. The name and owner must be nonempty UTF-8 without NUL.
// The native service validates authorization and current ownership. Cancellation
// does not undo a submitted revocation.
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

// lockAcquirer is the native lock submission boundary; completion is asynchronous.
type lockAcquirer interface {
	AcquireLock(string, ChannelType, string, bool, *uint64) int
}

func submitLockAcquisition(n lockAcquirer, ch Channel, name string, options AcquireLockOptions) (int, uint64) {
	var id uint64
	code := n.AcquireLock(ch.Name, ch.Type, name, options.Retry, &id)
	return code, id
}
