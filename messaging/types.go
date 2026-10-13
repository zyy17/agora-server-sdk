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
	// ErrClosed reports that a client or stream has been closed, or that its
	// event queue has been closed and drained.
	ErrClosed = errors.New("messaging: client or stream is closed")
	// ErrInvalidArgument is wrapped by errors caused by local input validation.
	// Use errors.Is to match it; the error text identifies the invalid field.
	ErrInvalidArgument = errors.New("messaging: invalid argument")
	// ErrUnavailable reports that a required native object or interface could
	// not be obtained.
	ErrUnavailable = errors.New("messaging: native interface unavailable")
)

// Error describes an immediate or asynchronous native SDK rejection.
// Use errors.As to inspect it. Context cancellation and wrapper validation are
// reported separately and are not converted to Error.
type Error struct {
	// Op identifies the attempted operation, such as "publish" or "login".
	Op string
	// Code is the native SDK error code without translation.
	Code ErrorCode
	// Details contains additional callback text when supplied by the SDK.
	// It is empty for errors without additional detail.
	Details string
}

// Error returns the operation, numeric code, and any additional detail.
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

// Config configures one independently owned RTM client.
// The zero value is incomplete: AppID and UserID are required. Native options
// and their referenced values must not be mutated while New is running.
type Config struct {
	// AppID identifies the Agora project. It must be nonempty UTF-8 without NUL;
	// the native SDK validates its format and project permissions.
	AppID string
	// UserID is the client's RTM identity within AppID. It must be
	// nonempty UTF-8 without NUL; native identity restrictions still apply.
	UserID string
	// Identity selects string or numeric SDK identity interpretation. Zero uses
	// string IDs, except that Native.UseStringUserId is honored when Native is set.
	Identity IdentityMode
	// Connection configures transport and native presence/reconnection timers.
	// Nil preserves the native binding defaults or Native settings.
	Connection *ConnectionConfig
	// Log, Proxy, Encryption, and Private replace the corresponding Native
	// configuration when non-nil. Values are copied during New.
	Log        *LogConfig
	Proxy      *ProxyConfig
	Encryption *EncryptionConfig
	Private    *PrivateConfig
	// Parameters is a JSON object applied at creation; empty preserves Native.
	Parameters string
	// ResultEvents adds EventRequestResult notifications for every asynchronous
	// result, including late/repeated results after cancellation or timeout.
	// These events share the bounded queue; request waits never rely on it.
	ResultEvents bool
	// EventBuffer is the capacity of the event queue. Zero selects 128 events;
	// explicit capacities must be 1 through 65536. Overflow drops the newest event.
	EventBuffer int
	// RequestTimeout bounds each asynchronous request, including waiting to
	// initiate it. Zero selects 15 seconds; negative durations are invalid.
	// A shorter caller deadline wins. It does not bound Next, WithNative,
	// SetParameters, NewStream, or Close, nor interrupt a running native call.
	RequestTimeout time.Duration
	// Native supplies advanced configuration such as logging, region, proxy,
	// and encryption. Nil uses native.NewRtmConfig defaults. New overrides
	// AppId, UserId, and EventHandler. Identity and the typed configuration above
	// override only their explicitly supplied settings. Start with
	// native.NewRtmConfig when customizing the remaining settings.
	Native *native.RtmConfig
}

// ChannelType is the native channel-kind enumeration.
// Supported kinds depend on the operation; the zero value is not a valid target.
type ChannelType = native.RtmChannelType

const (
	// MessageChannel identifies a message channel for publication, presence,
	// metadata, locks, and history.
	MessageChannel = native.RtmChannelTypeMESSAGE
	// StreamChannel identifies a stream channel for presence, metadata, and
	// locks. Topic messaging uses a Stream; history does not support this kind.
	StreamChannel = native.RtmChannelTypeSTREAM
	// UserChannel identifies a direct-user channel for history.
	// Use Client.PublishToUser to send direct messages.
	UserChannel = native.RtmChannelTypeUSER
)

// MessageType is the native payload-kind enumeration.
type MessageType = native.RtmMessageType

const (
	// TextMessage identifies a UTF-8 text payload.
	TextMessage = native.RtmMessageTypeSTRING
	// BinaryMessage identifies an arbitrary byte payload.
	BinaryMessage = native.RtmMessageTypeBINARY
)

// Channel identifies a target by name and native channel kind.
// Presence, channel metadata, and locks accept MessageChannel or StreamChannel.
// Client.History accepts MessageChannel or UserChannel.
type Channel struct {
	// Name is the channel name, or user-channel identifier for UserChannel.
	// Inputs must be nonempty UTF-8 without NUL.
	Name string
	// Type selects the target kind. Its zero value is invalid for operations.
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

// PublishOptions controls message-channel and direct-user publications.
// The zero value sends a binary message without requesting history storage.
type PublishOptions struct {
	// Text selects TextMessage and requires valid UTF-8. False selects
	// BinaryMessage. Both formats preserve embedded NUL bytes.
	Text bool
	// CustomType is optional application-defined message metadata. It must be
	// UTF-8 without NUL; the native SDK enforces its size limit.
	CustomType string
	// StoreInHistory requests server-side history storage for this message.
	// The project must have history access; this flag does not bypass service
	// permissions, retention, or quotas.
	StoreInHistory bool
}

// SubscribeOptions controls a message-channel subscription.
// The zero value enables messages and disables the other event families,
// which differs from some native defaults.
type SubscribeOptions struct {
	// DisableMessages permits presence/metadata/lock-only subscriptions. False
	// preserves the default message subscription (native withMessage=true).
	DisableMessages bool
	// Presence requests channel presence events.
	Presence bool
	// Metadata requests channel metadata events.
	Metadata bool
	// Locks requests channel lock events.
	Locks bool
	// Quiet requests subscription without notifying other users of this
	// client's subscribe, unsubscribe, or presence-state changes.
	Quiet bool
}

// PresenceOptions selects one page of an online-user query.
// User IDs are included unless ExcludeUserID is true.
type PresenceOptions struct {
	// ExcludeUserID forwards includeUserId=false for queries that do not need
	// identities. The pinned C callback does not expose total occupancy; do not
	// infer a channel-wide count from the length of the returned Users slice.
	ExcludeUserID bool
	// IncludeState requests each user's channel-scoped state in the result.
	IncludeState bool
	// Page is the opaque cursor returned as PresencePage.NextPage.
	// Use an empty string for the first page and pass later cursors unchanged.
	Page string
}

// UserState contains a user's channel-scoped presence state.
// Returned maps are Go-owned copies. Presence is not durable metadata storage.
type UserState struct {
	// UserID identifies the user whose state is represented.
	UserID string
	// State maps state keys to values. It may be empty when state was not
	// requested or none was supplied by the SDK.
	State map[string]string
}

// PresencePage is one page returned by Client.OnlineUsers.
// It may describe only part of a channel; successive pages and live events do
// not form an atomic snapshot.
type PresencePage struct {
	// Users contains the users returned on this page, with Go-owned state.
	Users []UserState
	// NextPage is the opaque cursor for the next request. An empty string
	// means the SDK reported no further page.
	NextPage string
}

// MetadataItem is one channel or user metadata entry.
// For unconditional writes, explicitly set Revision to -1. Zero is forwarded
// unchanged and is not interpreted as the native unconditional-write default.
type MetadataItem struct {
	// Key identifies the item. Write inputs require unique, nonempty UTF-8
	// keys without NUL within each Metadata value.
	Key string
	// Value is the item's value. Write inputs must be UTF-8 without NUL;
	// an empty value is allowed by local validation.
	Value string
	// AuthorUserID is the SDK-reported last modifying user, when recorded.
	// Write inputs may be empty but must be UTF-8 without NUL.
	AuthorUserID string
	// Revision is the per-item revision or write precondition.
	// Use -1 to omit its revision check; zero is passed to the SDK unchanged.
	Revision int64
	// UpdatedAt is the SDK-reported modification timestamp, when recorded.
	// The wrapper preserves the native timestamp value without unit conversion.
	UpdatedAt int64
}

// Metadata is a channel or user metadata collection with a whole-set revision.
// Write operations require at least one item. For unconditional writes, set both
// Revision and each MetadataItem.Revision to -1 explicitly.
type Metadata struct {
	// Revision is the collection's major revision or write precondition.
	// Use -1 to omit its revision check; the zero value is not rewritten.
	Revision int64
	// Items contains the entries to return, set, update, or remove.
	// Returned entries are Go-owned copies.
	Items []MetadataItem
}

// MetadataOptions controls metadata writes.
// The zero value requests neither timestamp nor author recording and uses no lock.
type MetadataOptions struct {
	// RecordTimestamp asks the server to record the modification timestamp.
	RecordTimestamp bool
	// RecordUserID asks the server to record the modifying user ID.
	RecordUserID bool
	// LockName optionally names a lock protecting a channel metadata write.
	// It must be UTF-8 without NUL. User metadata writes reject a nonempty value;
	// the service validates lock ownership.
	LockName string
}

// Lock describes a server-side channel lock.
// Creating a lock definition and acquiring its ownership are separate operations.
type Lock struct {
	// Name identifies the lock within its channel.
	Name string
	// Owner identifies the current owner when reported. The native SDK
	// provides this field for lock queries and snapshot events; other event
	// kinds need not contain an owner.
	Owner string
	// TTL is the lock's native time-to-live converted from seconds.
	// It governs retained ownership after disconnection, not the
	// lifetime of the lock definition.
	TTL time.Duration
}

// HistoryOptions selects a newest-to-oldest history page.
// Timestamps retain native SDK units. Nonzero bounds must satisfy Start >= End;
// negative timestamps are invalid. Native interval and retention limits still apply.
type HistoryOptions struct {
	// Start is the exclusive newer bound or previous HistoryPage.NextStart.
	// Zero leaves the bound unset. Preserve SDK cursors without changing units.
	Start int64
	// End is the inclusive older bound. Zero leaves the bound unset.
	End int64
	// Count limits the page size. Zero selects 100; explicit values must
	// be 1 through 100.
	Count int
}

// HistoryMessage is a Go-owned copy of a stored publication.
type HistoryMessage struct {
	// Type identifies text or binary payload interpretation.
	Type MessageType
	// Publisher is the publishing user's ID.
	Publisher string
	// Data contains the full payload, including any embedded NUL bytes.
	Data []byte
	// CustomType is the application-defined type supplied at publication.
	CustomType string
	// Timestamp is when the RTM server received the message, in native units.
	// The wrapper does not convert it to time.Time or a media timestamp.
	Timestamp uint64
}

// HistoryPage is one page of stored messages returned by Client.History.
// A page is limited by the query and service retention; it is not a delivery log
// or a guarantee that recipient applications processed the messages.
type HistoryPage struct {
	// Messages contains Go-owned copies of the messages returned by the SDK.
	Messages []HistoryMessage
	// NextStart is the native cursor for the next, older page. Zero means no
	// further page. Use it as HistoryOptions.Start without changing units,
	// checking that it fits int64 before conversion.
	NextStart uint64
}

// TopicSubscription reports per-publisher results from Stream.SubscribeTopic.
// Inspect both lists even when the accompanying error is non-nil; successful
// subscriptions are not rolled back when other publishers fail.
type TopicSubscription struct {
	// Succeeded contains publisher IDs whose subscriptions succeeded.
	Succeeded []string
	// Failed contains publisher IDs whose subscriptions failed.
	Failed []string
}

// EventKind identifies which payload in Event is populated.
type EventKind string

const (
	// EventRequestResult identifies an optional Event.RequestResult payload.
	EventRequestResult EventKind = "request_result"
	// EventMessage identifies an Event.Message payload.
	EventMessage EventKind = "message"
	// EventPresence identifies an Event.Presence payload.
	EventPresence EventKind = "presence"
	// EventTopic identifies an Event.Topic payload.
	EventTopic EventKind = "topic"
	// EventLock identifies an Event.Lock payload.
	EventLock EventKind = "lock"
	// EventStorage identifies an Event.Storage payload.
	EventStorage EventKind = "storage"
	// EventToken identifies an Event.Token payload.
	EventToken EventKind = "token"
	// EventTokenExpiring identifies a token-expiry notice in Event.Channel.
	// It does not populate Event.Token.
	EventTokenExpiring EventKind = "token_expiring"
	// EventLink identifies an Event.Link payload.
	EventLink EventKind = "link"
)

// Event is one native notification or optional request result copied into the client queue.
// Kind identifies its payload; unrelated pointer fields are nil. EventTokenExpiring
// uses Channel rather than a pointer payload. Nested data is Go-owned and remains
// valid after callbacks return and after Client.Close. A full queue can drop any
// event kind, including token and connection notifications.
type Event struct {
	// RequestResult is populated for EventRequestResult when ResultEvents is enabled.
	RequestResult *RequestResult
	// Kind identifies the notification and the payload to inspect.
	Kind EventKind
	// Message is populated for EventMessage.
	Message *Message
	// Presence is populated for EventPresence.
	Presence *PresenceEvent
	// Topic is populated for EventTopic.
	Topic *TopicEvent
	// Lock is populated for EventLock.
	Lock *LockEvent
	// Storage is populated for EventStorage.
	Storage *StorageEvent
	// Link is populated for EventLink.
	Link *LinkEvent
	// Token is populated for EventToken.
	Token *TokenEvent
	// Channel is the native channel identifier for EventTokenExpiring.
	// Use it to determine which client or stream token needs renewal; inspect
	// other event payloads for their channel information.
	Channel string
}

// Message is a received message-channel, direct-user, or stream-topic publication.
// Its payload is independent of the native callback's storage.
type Message struct {
	// Channel identifies the native destination associated with the message.
	Channel Channel
	// Topic identifies a stream-channel topic when supplied by the SDK.
	Topic string
	// Type identifies text or binary payload interpretation.
	Type MessageType
	// Data contains the complete, Go-owned payload, including NUL bytes.
	Data []byte
	// Publisher is the sending user's ID.
	Publisher string
	// CustomType is the application-defined type supplied by the publisher.
	CustomType string
}

// PresenceEvent describes a presence snapshot, interval update, or individual
// change. Fields are populated according to the native event Type. Updates and
// snapshots must not be assumed to be complete channel state: the wrapper exposes
// only the state carried by this notification, and queued events can be dropped.
type PresenceEvent struct {
	// Type is the unmodified native RTM_PRESENCE_EVENT_TYPE value.
	Type PresenceEventType
	// Channel identifies the message or stream channel.
	Channel Channel
	// Publisher identifies the user associated with an individual change.
	Publisher string
	// State contains the state items carried by an individual notification.
	// It is a Go-owned map and need not contain all of the user's state.
	State map[string]string
	// Joined lists users reported as joining in an interval notification.
	Joined []string
	// Left lists users reported as leaving in an interval notification.
	Left []string
	// TimedOut lists users reported as timing out in an interval notification.
	TimedOut []string
	// Updated contains state changes supplied in an interval notification.
	Updated []UserState
	// Snapshot contains the users supplied in a snapshot notification.
	// Use Client.OnlineUsers and its cursor when querying paginated membership.
	Snapshot []UserState
}

// TopicPublisher describes a publisher advertised in a topic event.
type TopicPublisher struct {
	// UserID identifies the topic publisher.
	UserID string
	// Metadata is the publisher metadata supplied when joining the topic.
	Metadata string
}

// TopicInfo describes a topic and the publishers included in a notification.
type TopicInfo struct {
	// Name identifies the topic within its stream channel.
	Name string
	// Publishers contains the publisher records supplied by the SDK.
	Publishers []TopicPublisher
}

// TopicEvent describes a native topic snapshot or publisher change.
// Interpret Topics according to Type; an update is not necessarily a full snapshot.
type TopicEvent struct {
	// Type is the unmodified native RTM_TOPIC_EVENT_TYPE value.
	Type TopicEventType
	// Channel identifies the stream channel.
	Channel string
	// Publisher is the user associated with the topic event.
	Publisher string
	// Topics contains the topic and publisher information in the notification.
	Topics []TopicInfo
}

// LockEvent describes a lock snapshot or change in a channel.
// Events can be dropped; do not treat the event queue alone as proof of ownership.
type LockEvent struct {
	// Type is the unmodified native RTM_LOCK_EVENT_TYPE value.
	Type LockEventType
	// Channel identifies the message or stream channel.
	Channel Channel
	// Locks contains the lock records supplied by the SDK.
	// Owner is defined for snapshot records but may be absent in other events.
	Locks []Lock
}

// StorageEvent describes a channel or user metadata snapshot or change.
// Interpret Metadata according to Type; a change need not contain the entire store.
type StorageEvent struct {
	// Type is the unmodified native RTM_STORAGE_EVENT_TYPE value.
	Type StorageEventType
	// StorageType is the native RTM_STORAGE_TYPE identifying user or channel storage.
	StorageType StorageType
	// ChannelType identifies the channel kind for channel-storage events.
	ChannelType ChannelType
	// Target is the affected channel name or user ID, according to StorageType.
	Target string
	// Metadata contains the Go-owned metadata carried by this notification.
	Metadata Metadata
}

// LinkEvent describes a native connection-state transition.
// Numeric fields preserve the linked SDK's values rather than defining a separate
// Go state machine. The application is responsible for any recovery decisions.
type LinkEvent struct {
	// CurrentState is the new native RTM_LINK_STATE value.
	CurrentState LinkState
	// PreviousState is the preceding native RTM_LINK_STATE value.
	PreviousState LinkState
	// ServiceType is the native RTM_SERVICE_TYPE for the affected service.
	ServiceType ServiceType
	// Operation is the native RTM_LINK_OPERATION that caused the transition.
	Operation LinkOperation
	// ReasonCode is the native RTM_LINK_STATE_CHANGE_REASON value.
	ReasonCode LinkReason
	// Reason contains the SDK's explanatory text.
	Reason string
	// AffectedChannels lists the channels affected by this transition.
	AffectedChannels []string
	// UnrestoredChannels lists channels the SDK could not restore.
	UnrestoredChannels []string
	// Resumed reports the native isResumed flag for connection recovery.
	Resumed bool
	// Timestamp is the native event timestamp without unit conversion.
	Timestamp uint64
}

// TokenEvent describes token expiry or permission changes reported by the SDK.
// The package does not obtain or renew tokens automatically. Renew the matching
// client or stream token through a trusted token service, and track expiry outside
// the event queue because overflow can drop these notifications.
type TokenEvent struct {
	// Type is the unmodified native RTM_TOKEN_EVENT_TYPE value.
	Type TokenEventType
	// Reason contains the SDK's explanatory text.
	Reason string
	// MessageChannels lists the message channels affected by the token event.
	MessageChannels []string
	// Timestamp is the native event timestamp without unit conversion.
	Timestamp uint64
}

func (o SubscribeOptions) nativeOptions() *native.SubscribeOptions {
	return &native.SubscribeOptions{WithMessage: !o.DisableMessages, WithPresence: o.Presence, WithMetadata: o.Metadata, WithLock: o.Locks, BeQuiet: o.Quiet}
}

func (o PresenceOptions) nativeOptions() *native.GetOnlineUsersOptions {
	return &native.GetOnlineUsersOptions{IncludeUserId: !o.ExcludeUserID, IncludeState: o.IncludeState, Page: o.Page}
}
