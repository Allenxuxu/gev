package ws

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/Allenxuxu/ringbuffer"
	"github.com/Allenxuxu/toolkit/convert"
	"github.com/gobwas/httphead"
)

// MessageType represents the type of a WebSocket message.
// See https://tools.ietf.org/html/rfc6455#section-5.6
type MessageType int

const (
	// MessageText Text
	MessageText MessageType = iota + 1
	// MessageBinary Binary
	MessageBinary
)

type handshakeHeader [2]HandshakeHeader

func (hs handshakeHeader) WriteTo(w io.Writer) (n int64, err error) {
	for i := 0; i < len(hs) && err == nil; i++ {
		if h := hs[i]; h != nil {
			var m int64
			m, err = h.WriteTo(w)
			n += m
		}
	}
	return n, err
}

// Handshake represents handshake result.
type Handshake struct {
	// Protocol is the subprotocol selected during handshake.
	Protocol string

	// Extensions is the list of negotiated extensions.
	Extensions []httphead.Option
}

// Upgrader contains options for upgrading connection to websocket.
type Upgrader struct {
	// Protocol is a select function that is used to select subprotocol
	// from list requested by client. If this field is set, then the first matched
	// protocol is sent to a client as negotiated.
	//
	// The argument is only valid until the callback returns.
	Protocol func([]byte) bool

	// ProtocolCustrom allow user to parse Sec-WebSocket-Protocol header manually.
	// Note that returned bytes must be valid until Upgrade returns.
	// If ProtocolCustom is set, it used instead of Protocol function.
	ProtocolCustom func([]byte) (string, bool)

	// Extension is a select function that is used to select extensions
	// from list requested by client. If this field is set, then the all matched
	// extensions are sent to a client as negotiated.
	//
	// The argument is only valid until the callback returns.
	//
	// According to the RFC6455 order of extensions passed by a client is
	// significant. That is, returning true from this function means that no
	// other extension with the same name should be checked because server
	// accepted the most preferable extension right now:
	// "Note that the order of extensions is significant.  Any interactions between
	// multiple extensions MAY be defined in the documents defining the extensions.
	// In the absence of such definitions, the interpretation is that the header
	// fields listed by the client in its request represent a preference of the
	// header fields it wishes to use, with the first options listed being most
	// preferable."
	Extension func(httphead.Option) bool

	// ExtensionCustorm allow user to parse Sec-WebSocket-Extensions header manually.
	// Note that returned options should be valid until Upgrade returns.
	// If ExtensionCustom is set, it used instead of Extension function.
	ExtensionCustom func([]byte, []httphead.Option) ([]httphead.Option, bool)

	// Header is an optional HandshakeHeader instance that could be used to
	// write additional headers to the handshake response.
	//
	// It used instead of any key-value mappings to avoid allocations in user
	// land.
	//
	// Note that if present, it will be written in any result of handshake.
	Header HandshakeHeader

	// OnRequest is a callback that will be called after request line
	// successful parsing.
	//
	// The arguments are only valid until the callback returns.
	//
	// If returned error is non-nil then connection is rejected and response is
	// sent with appropriate HTTP error code and body set to error message.
	//
	// RejectConnectionError could be used to get more control on response.
	OnRequest func(uri []byte) error

	// OnHost is a callback that will be called after "Host" header successful
	// parsing.
	//
	// It is separated from OnHeader callback because the Host header must be
	// present in each request since HTTP/1.1. Thus Host header is non-optional
	// and required for every WebSocket handshake.
	//
	// The arguments are only valid until the callback returns.
	//
	// If returned error is non-nil then connection is rejected and response is
	// sent with appropriate HTTP error code and body set to error message.
	//
	// RejectConnectionError could be used to get more control on response.
	OnHost func(host []byte) error

	// OnHeader is a callback that will be called after successful parsing of
	// header, that is not used during WebSocket handshake procedure. That is,
	// it will be called with non-websocket headers, which could be relevant
	// for application-level logic.
	//
	// The arguments are only valid until the callback returns.
	//
	// If returned error is non-nil then connection is rejected and response is
	// sent with appropriate HTTP error code and body set to error message.
	//
	// RejectConnectionError could be used to get more control on response.
	OnHeader func(key, value []byte) error

	// OnBeforeUpgrade is a callback that will be called before sending
	// successful upgrade response.
	//
	// Setting OnBeforeUpgrade allows user to make final application-level
	// checks and decide whether this connection is allowed to successfully
	// upgrade to WebSocket.
	//
	// It must return non-nil either HandshakeHeader or error and never both.
	//
	// If returned error is non-nil then connection is rejected and response is
	// sent with appropriate HTTP error code and body set to error message.
	//
	// RejectConnectionError could be used to get more control on response.
	OnBeforeUpgrade func() (header HandshakeHeader, err error)
}

// Upgrade zero-copy upgrades connection to WebSocket. It interprets given conn
// as connection with incoming HTTP Upgrade request.
//
// It is a caller responsibility to manage i/o timeouts on conn.
//
// Non-nil error means that request for the WebSocket upgrade is invalid or
// malformed and usually connection should be closed.
// Even when error is non-nil Upgrade will write appropriate response into
// connection in compliance with RFC.
func (u *Upgrader) Upgrade(in *ringbuffer.RingBuffer) (out []byte, hs Handshake, err error) {
	// headerSeen constants helps to report whether or not some header was seen
	// during reading request bytes.
	const (
		headerSeenHost = 1 << iota
		headerSeenUpgrade
		headerSeenConnection
		headerSeenSecVersion
		headerSeenSecKey

		// headerSeenAll is the value that we expect to receive at the end of
		// headers read/parse loop.
		headerSeenAll = 0 |
			headerSeenHost |
			headerSeenUpgrade |
			headerSeenConnection |
			headerSeenSecVersion |
			headerSeenSecKey
	)

	first, end := in.PeekAll()
	var data []byte
	if index := bytes.Index(first, []byte("\r\n\r\n")); index != -1 {
		data = make([]byte, index+4)
		if _, err = in.Read(data); err != nil {
			return
		}
	} else if len(end) > 0 {
		if index := bytes.Index(end, []byte("\r\n\r\n")); index != -1 {
			data = make([]byte, index+4)
			if _, err = in.Read(data); err != nil {
				return
			}
		}
	}

	lines := bytes.Split(data, []byte("\r\n"))
	if len(lines) == 0 {
		err = errors.New("len(lines) = 0")
		return
	}

	// Parse request line data like HTTP version, uri and method.
	req, err := httpParseRequestLine(lines[0])
	if err != nil {
		return
	}

	// Prepare stack-based handshake header list.
	header := handshakeHeader{
		0: u.Header,
	}

	// An HTTP/1.1 or higher GET request, including a "Request-URI".
	//
	// Even if RFC says "1.1 or higher" without mentioning the part of the
	// version, we apply it only to minor part.
	switch {
	case req.major != 1 || req.minor < 1:
		// Abort processing the whole request because we do not even know how
		// to actually parse it.
		err = ErrHandshakeBadProtocol
	case convert.BytesToString(req.method) != http.MethodGet:
		err = ErrHandshakeBadMethod
	default:
		if onRequest := u.OnRequest; onRequest != nil {
			err = onRequest(req.uri)
		}
	}
	// Start headers read/parse loop.
	var (
		// headerSeen reports which header was seen by setting corresponding
		// bit on.
		headerSeen byte
		nonce      = make([]byte, nonceSize)
	)
	for i := 1; i < len(lines); i++ {
		if len(lines[i]) == 0 {
			// Blank line, no more lines to read.
			break
		}

		k, v, ok := httpParseHeaderLine(lines[i])
		if !ok {
			err = ErrMalformedRequest
			break
		}

		switch convert.BytesToString(k) {
		case headerHostCanonical:
			headerSeen |= headerSeenHost
			if onHost := u.OnHost; onHost != nil {
				err = onHost(v)
			}
		case headerUpgradeCanonical:
			headerSeen |= headerSeenUpgrade
			if !bytes.Equal(v, specHeaderValueUpgrade) {
				err = ErrHandshakeBadUpgrade
			}
		case headerConnectionCanonical:
			headerSeen |= headerSeenConnection
			if !bytes.Equal(v, specHeaderValueConnection) {
				err = ErrHandshakeBadConnection
			}
		case headerSecVersionCanonical:
			headerSeen |= headerSeenSecVersion
			if !bytes.Equal(v, specHeaderValueSecVersion) {
				err = ErrHandshakeUpgradeRequired
			}
		case headerSecKeyCanonical:
			headerSeen |= headerSeenSecKey
			if len(v) != nonceSize {
				err = ErrHandshakeBadSecKey
			} else {
				copy(nonce[:], v)
			}
		case headerSecProtocolCanonical:
			if custom, check := u.ProtocolCustom, u.Protocol; hs.Protocol == "" && (custom != nil || check != nil) {
				var ok bool
				if custom != nil {
					hs.Protocol, ok = custom(v)
				} else {
					hs.Protocol, ok = btsSelectProtocol(v, check)
				}
				if !ok {
					err = ErrMalformedRequest
				}
			}
		case headerSecExtensionsCanonical:
			if custom, check := u.ExtensionCustom, u.Extension; custom != nil || check != nil {
				var ok bool
				if custom != nil {
					hs.Extensions, ok = custom(v, hs.Extensions)
				} else {
					hs.Extensions, ok = btsSelectExtensions(v, hs.Extensions, check)
				}
				if !ok {
					err = ErrMalformedRequest
				}
			}
		default:
			if onHeader := u.OnHeader; onHeader != nil {
				err = onHeader(k, v)
			}
		}
	}
	switch {
	case err == nil && headerSeen != headerSeenAll:
		switch {
		case headerSeen&headerSeenHost == 0:
			err = ErrHandshakeBadHost
		case headerSeen&headerSeenUpgrade == 0:
			err = ErrHandshakeBadUpgrade
		case headerSeen&headerSeenConnection == 0:
			err = ErrHandshakeBadConnection
		case headerSeen&headerSeenSecVersion == 0:
			err = ErrHandshakeBadSecVersion
		case headerSeen&headerSeenSecKey == 0:
			err = ErrHandshakeBadSecKey
		default:
			panic("unknown headers state")
		}

	case err == nil && u.OnBeforeUpgrade != nil:
		header[1], err = u.OnBeforeUpgrade()
	}
	if err != nil {
		var code int
		if rej, ok := err.(*rejectConnectionError); ok {
			code = rej.code
			header[1] = rej.header
		}
		if code == 0 {
			code = http.StatusInternalServerError
		}
		out = httpWriteResponseError(err, code, header.WriteTo)
		return
	}

	out = httpWriteResponseUpgrade(nonce, hs, header.WriteTo)
	return
}
