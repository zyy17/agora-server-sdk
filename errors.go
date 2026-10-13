package agorasdk

import (
	"errors"
	"fmt"
)

var (
	// ErrClosed reports an operation on a closed high-level resource.
	ErrClosed = errors.New("agora: closed")
	// ErrServiceActive reports that another Service owns the process-wide RTC runtime.
	ErrServiceActive = errors.New("agora: a service already owns the process-wide native runtime")
	// ErrInvalidConfig is wrapped by argument-validation errors. Use errors.Is to match it.
	ErrInvalidConfig = errors.New("agora: invalid configuration")
	// ErrUnsupported reports an unavailable build feature or unsupported wrapper operation.
	ErrUnsupported = errors.New("agora: unsupported operation")
	// ErrNotPublishing reports a role or publication mode incompatible with the requested send.
	ErrNotPublishing = errors.New("agora: publishing is disabled for this media type")
)

// Error preserves an operation name and its native SDK or binding error code.
// Use errors.As to inspect it. Interpret Code together with Op and the pinned
// SDK version; the wrapper does not normalize native error-code families.
// Error does not include supplied tokens or credentials.
type Error struct {
	// Op names the failed operation.
	Op string
	// Code is the nonzero native SDK, binding, or connection failure code.
	Code int
}

// Error formats the operation and numeric code without credentials.
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
