// Package agorasdk provides owned, context-aware RTC sessions on Agora Server SDK.
//
// Create one Service per process, Join channels with independent identities and
// tokens, and defer Close. Send methods publish automatically. Receive streams own
// their bytes and never block native callbacks. A full stream drops its oldest
// item; inspect Stream.Stats for dropped counts. Join uses context, while a
// successfully returned Session lives until explicitly closed.
package agorasdk
