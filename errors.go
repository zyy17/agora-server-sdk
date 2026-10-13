package agorasdk

import (
	"errors"
	"fmt"
)

var (
	ErrClosed        = errors.New("agora: closed")
	ErrServiceActive = errors.New("agora: a service already owns the process-wide native runtime")
	ErrInvalidConfig = errors.New("agora: invalid configuration")
	ErrUnsupported   = errors.New("agora: unsupported operation")
	ErrNotPublishing = errors.New("agora: publishing is disabled for this media type")
)

// Error preserves the native code without exposing tokens or credentials.
type Error struct {
	Op   string
	Code int
}

func (e *Error) Error() string { return fmt.Sprintf("agora: %s failed (native code %d)", e.Op, e.Code) }

func check(op string, code int) error {
	if code == 0 {
		return nil
	}
	return &Error{Op: op, Code: code}
}

func invalid(field, problem string) error {
	return fmt.Errorf("%w: %s %s", ErrInvalidConfig, field, problem)
}
