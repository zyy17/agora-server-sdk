// Package token generates Agora RTC AccessToken2 values locally for string user
// accounts. It wraps Agora's Go token builder and has no native SDK or network
// dependency during token generation.
//
// # Scope and identity
//
// [RTC] authorizes one exact channel and string user account from [Config].
// Use the same identity when joining the RTC channel. This package always uses
// the string-account builder, including for digit-only account strings. It does
// not expose numeric-UID token generation, generate RTM login tokens, or combine
// RTC and RTM privileges. Use the appropriate Agora token builder for those
// workflows.
//
// # Privileges and expiry
//
// The zero [Role] is [Publisher], which includes joining and publishing audio,
// video, and data streams. [Subscriber] includes only joining. Server-side
// enforcement of publishing restrictions depends on the Agora project's co-host
// authentication setting; selecting Subscriber alone is not a security boundary.
//
// [Config.Lifetime] is a duration from creation time. Zero selects one hour;
// nonzero values must be from one second through 24 hours. The duration is
// truncated to integer seconds and applied equally to the token and every
// included privilege. Generating a replacement token does not renew an existing
// connection; the application must deliver it and call the client's renewal API.
//
// # Credentials
//
// Generate tokens only on a trusted server. Keep the App Certificate out of
// client applications, source control, logs, and error reports. Treat generated
// tokens as credentials, return them only to the intended client over a secure
// channel, and authorize that client's requested channel, identity, and role
// before issuing one. Successful local generation verifies configuration shape;
// it does not prove that the App ID or certificate belongs to a valid project.
package token
