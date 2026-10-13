// Package messaging provides owned, context-aware Agora RTM clients. Native SDK
// callbacks are copied into a bounded event channel; application code never runs
// on an SDK callback thread. Canceling a request stops waiting, not the server
// operation. A canceled publish or metadata update may still take effect.
package messaging

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	native "github.com/zyy17/agora-server-sdk/agora/rtm"
)

var (
	ErrClosed          = errors.New("messaging: client or stream is closed")
	ErrInvalidArgument = errors.New("messaging: invalid argument")
	ErrUnavailable     = errors.New("messaging: native interface unavailable")
)

// ErrorCode is an unmodified native RTM error code.
type ErrorCode int

// Error is an SDK rejection, either immediate or asynchronous. Codes preserve
// the SDK's numeric value; context cancellation remains context.Canceled or
// context.DeadlineExceeded instead.
type Error struct {
	Op      string
	Code    ErrorCode
	Details string
}

func (e *Error) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("messaging: %s: RTM error %d: %s", e.Op, e.Code, e.Details)
	}
	return fmt.Sprintf("messaging: %s: RTM error %d", e.Op, e.Code)
}

func sdkError(op string, code int) error {
	if code == 0 {
		return nil
	}
	return &Error{Op: op, Code: ErrorCode(code)}
}

func invalid(field string) error { return fmt.Errorf("%w: %s", ErrInvalidArgument, field) }

func validText(field, s string, required bool) error {
	if (required && s == "") || strings.IndexByte(s, 0) >= 0 || !utf8.ValidString(s) {
		return invalid(field)
	}
	return nil
}

// Config configures one independently owned RTM client. Zero EventBuffer and
// RequestTimeout select 128 events and 15 seconds. Native optionally supplies
// advanced configuration (logging, region, proxy, encryption); AppID, UserID,
// UseStringUserId and EventHandler are always set by this package. Native and its
// pointed-to values must not be mutated while New is running.
type Config struct {
	AppID          string
	UserID         string
	EventBuffer    int
	RequestTimeout time.Duration
	Native         *native.RtmConfig
}

type ChannelType = native.RtmChannelType

const (
	MessageChannel = native.RtmChannelTypeMESSAGE
	StreamChannel  = native.RtmChannelTypeSTREAM
	UserChannel    = native.RtmChannelTypeUSER
)

type MessageType = native.RtmMessageType

const (
	TextMessage   = native.RtmMessageTypeSTRING
	BinaryMessage = native.RtmMessageTypeBINARY
)

// Channel identifies a target channel. Presence, channel metadata and locks use
// MessageChannel or StreamChannel; History uses MessageChannel or UserChannel.
// Use PublishToUser to send direct messages.
type Channel struct {
	Name string
	Type ChannelType
}

func (c Channel) validate() error {
	if err := validText("channel name", c.Name, true); err != nil {
		return err
	}
	if c.Type != MessageChannel && c.Type != StreamChannel {
		return invalid("channel type")
	}
	return nil
}

type PublishOptions struct {
	Text           bool
	CustomType     string
	StoreInHistory bool
}

// SubscribeOptions always enables messages. Other event families are opt-in.
type SubscribeOptions struct {
	Presence bool
	Metadata bool
	Locks    bool
	Quiet    bool
}
type PresenceOptions struct {
	IncludeState bool
	Page         string
}
type UserState struct {
	UserID string
	State  map[string]string
}
type PresencePage struct {
	Users    []UserState
	NextPage string
}
type MetadataItem struct {
	Key, Value, AuthorUserID string
	Revision                 int64
	UpdatedAt                int64
}
type Metadata struct {
	Revision int64
	Items    []MetadataItem
}
type MetadataOptions struct {
	RecordTimestamp bool
	RecordUserID    bool
	LockName        string
}
type Lock struct {
	Name, Owner string
	TTL         time.Duration
}
type HistoryOptions struct {
	Start, End int64
	Count      int
}
type HistoryMessage struct {
	Type       MessageType
	Publisher  string
	Data       []byte
	CustomType string
	Timestamp  uint64
}
type HistoryPage struct {
	Messages  []HistoryMessage
	NextStart uint64
}
type TopicSubscription struct{ Succeeded, Failed []string }

// EventKind identifies the populated field of Event.
type EventKind string

const (
	EventMessage       EventKind = "message"
	EventPresence      EventKind = "presence"
	EventTopic         EventKind = "topic"
	EventLock          EventKind = "lock"
	EventStorage       EventKind = "storage"
	EventToken         EventKind = "token"
	EventTokenExpiring EventKind = "token_expiring"
	EventLink          EventKind = "link"
)

type Event struct {
	Kind     EventKind
	Message  *Message
	Presence *PresenceEvent
	Topic    *TopicEvent
	Lock     *LockEvent
	Storage  *StorageEvent
	Link     *LinkEvent
	Token    *TokenEvent
	Channel  string
}
type Message struct {
	Channel               Channel
	Topic                 string
	Type                  MessageType
	Data                  []byte
	Publisher, CustomType string
}
type PresenceEvent struct {
	Type                   int
	Channel                Channel
	Publisher              string
	State                  map[string]string
	Joined, Left, TimedOut []string
	Updated, Snapshot      []UserState
}
type TopicPublisher struct{ UserID, Metadata string }
type TopicInfo struct {
	Name       string
	Publishers []TopicPublisher
}
type TopicEvent struct {
	Type      int
	Channel   string
	Publisher string
	Topics    []TopicInfo
}
type LockEvent struct {
	Type    int
	Channel Channel
	Locks   []Lock
}
type StorageEvent struct {
	Type        int
	StorageType int
	ChannelType ChannelType
	Target      string
	Metadata    Metadata
}
type LinkEvent struct {
	CurrentState, PreviousState, ServiceType, Operation, ReasonCode int
	Reason                                                          string
	AffectedChannels, UnrestoredChannels                            []string
	Resumed                                                         bool
	Timestamp                                                       uint64
}

type TokenEvent struct {
	Type            int
	Reason          string
	MessageChannels []string
	Timestamp       uint64
}
