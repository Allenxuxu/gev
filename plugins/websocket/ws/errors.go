package ws

import (
	"fmt"
	"net/http"
)

// ProtocolError describes error during checking/parsing websocket frames or
// headers.
type ProtocolError string

// Error implements error interface.
func (p ProtocolError) Error() string { return string(p) }

// Errors used by the protocol checkers.
var (
	ErrProtocolStatusCodeNotInUse         = ProtocolError("status code is not in use")
	ErrProtocolStatusCodeApplicationLevel = ProtocolError("status code is only application level")
	ErrProtocolStatusCodeNoMeaning        = ProtocolError("status code has no meaning yet")
	ErrProtocolStatusCodeUnknown          = ProtocolError("status code is not defined in spec")
	ErrProtocolInvalidUTF8                = ProtocolError("invalid utf8 sequence in close reason")
)

// Errors used by both client and server when preparing WebSocket handshake.
var (
	ErrHandshakeBadProtocol = RejectConnectionError(
		RejectionStatus(http.StatusHTTPVersionNotSupported),
		RejectionReason("handshake error: bad HTTP protocol version"),
	)
	ErrHandshakeBadMethod = RejectConnectionError(
		RejectionStatus(http.StatusMethodNotAllowed),
		RejectionReason("handshake error: bad HTTP request method"),
	)
	ErrHandshakeBadHost = RejectConnectionError(
		RejectionStatus(http.StatusBadRequest),
		RejectionReason(fmt.Sprintf("handshake error: bad %q header", headerHost)),
	)
	ErrHandshakeBadUpgrade = RejectConnectionError(
		RejectionStatus(http.StatusBadRequest),
		RejectionReason(fmt.Sprintf("handshake error: bad %q header", headerUpgrade)),
	)
	ErrHandshakeBadConnection = RejectConnectionError(
		RejectionStatus(http.StatusBadRequest),
		RejectionReason(fmt.Sprintf("handshake error: bad %q header", headerConnection)),
	)
	ErrHandshakeBadSecAccept = RejectConnectionError(
		RejectionStatus(http.StatusBadRequest),
		RejectionReason(fmt.Sprintf("handshake error: bad %q header", headerSecAccept)),
	)
	ErrHandshakeBadSecKey = RejectConnectionError(
		RejectionStatus(http.StatusBadRequest),
		RejectionReason(fmt.Sprintf("handshake error: bad %q header", headerSecKey)),
	)
	ErrHandshakeBadSecVersion = RejectConnectionError(
		RejectionStatus(http.StatusBadRequest),
		RejectionReason(fmt.Sprintf("handshake error: bad %q header", headerSecVersion)),
	)
)

// ErrMalformedRequest is returned when HTTP request can not be parsed.
var ErrMalformedRequest = RejectConnectionError(
	RejectionStatus(http.StatusBadRequest),
	RejectionReason("malformed HTTP request"),
)

// ErrHandshakeUpgradeRequired is returned by Upgrader to indicate that
// connection is rejected because given WebSocket version is malformed.
//
// According to RFC6455:
// If this version does not match a version understood by the server, the
// server MUST abort the WebSocket handshake described in this section and
// instead send an appropriate HTTP error code (such as 426 Upgrade Required)
// and a |Sec-WebSocket-Version| header field indicating the version(s) the
// server is capable of understanding.
var ErrHandshakeUpgradeRequired = RejectConnectionError(
	RejectionStatus(http.StatusUpgradeRequired),
	RejectionHeader(HandshakeHeaderString(headerSecVersion+": 13\r\n")),
	RejectionReason(fmt.Sprintf("handshake error: bad %q header", headerSecVersion)),
)

// RejectOption represents an option used to control the way connection is
// rejected.
type RejectOption func(*rejectConnectionError)

// RejectionReason returns an option that makes connection to be rejected with
// given reason.
func RejectionReason(reason string) RejectOption {
	return func(err *rejectConnectionError) {
		err.reason = reason
	}
}

// RejectionStatus returns an option that makes connection to be rejected with
// given HTTP status code.
func RejectionStatus(code int) RejectOption {
	return func(err *rejectConnectionError) {
		err.code = code
	}
}

// RejectionHeader returns an option that makes connection to be rejected with
// given HTTP headers.
func RejectionHeader(h HandshakeHeader) RejectOption {
	return func(err *rejectConnectionError) {
		err.header = h
	}
}

// RejectConnectionError constructs an error that could be used to control the way
// handshake is rejected by Upgrader.
func RejectConnectionError(options ...RejectOption) error {
	err := new(rejectConnectionError)
	for _, opt := range options {
		opt(err)
	}
	return err
}

// rejectConnectionError represents a rejection of upgrade error.
//
// It can be returned by Upgrader's On* hooks to control the way WebSocket
// handshake is rejected.
type rejectConnectionError struct {
	reason string
	code   int
	header HandshakeHeader
}

// Error implements error interface.
func (r *rejectConnectionError) Error() string {
	return r.reason
}
