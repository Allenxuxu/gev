package ws

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"net/textproto"
	"strconv"

	"github.com/gobwas/httphead"
)

const (
	crlf          = "\r\n"
	colonAndSpace = ": "
)

const (
	textHeadUpgrade = "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"
)

var (
	textHeadBadRequest          = statusText(http.StatusBadRequest)
	textHeadInternalServerError = statusText(http.StatusInternalServerError)
	textHeadUpgradeRequired     = statusText(http.StatusUpgradeRequired)

	textTailErrHandshakeBadProtocol   = errorText(ErrHandshakeBadProtocol)
	textTailErrHandshakeBadMethod     = errorText(ErrHandshakeBadMethod)
	textTailErrHandshakeBadHost       = errorText(ErrHandshakeBadHost)
	textTailErrHandshakeBadUpgrade    = errorText(ErrHandshakeBadUpgrade)
	textTailErrHandshakeBadConnection = errorText(ErrHandshakeBadConnection)
	textTailErrHandshakeBadSecAccept  = errorText(ErrHandshakeBadSecAccept)
	textTailErrHandshakeBadSecKey     = errorText(ErrHandshakeBadSecKey)
	textTailErrHandshakeBadSecVersion = errorText(ErrHandshakeBadSecVersion)
	textTailErrUpgradeRequired        = errorText(ErrHandshakeUpgradeRequired)
)

var (
	headerHost          = "Host"
	headerUpgrade       = "Upgrade"
	headerConnection    = "Connection"
	headerSecVersion    = "Sec-WebSocket-Version"
	headerSecProtocol   = "Sec-WebSocket-Protocol"
	headerSecExtensions = "Sec-WebSocket-Extensions"
	headerSecKey        = "Sec-WebSocket-Key"
	headerSecAccept     = "Sec-WebSocket-Accept"

	headerHostCanonical          = textproto.CanonicalMIMEHeaderKey(headerHost)
	headerUpgradeCanonical       = textproto.CanonicalMIMEHeaderKey(headerUpgrade)
	headerConnectionCanonical    = textproto.CanonicalMIMEHeaderKey(headerConnection)
	headerSecVersionCanonical    = textproto.CanonicalMIMEHeaderKey(headerSecVersion)
	headerSecProtocolCanonical   = textproto.CanonicalMIMEHeaderKey(headerSecProtocol)
	headerSecExtensionsCanonical = textproto.CanonicalMIMEHeaderKey(headerSecExtensions)
	headerSecKeyCanonical        = textproto.CanonicalMIMEHeaderKey(headerSecKey)
)

var (
	specHeaderValueUpgrade    = []byte("websocket")
	specHeaderValueConnection = []byte("Upgrade")
	//specHeaderValueConnectionLower = []byte("upgrade")
	specHeaderValueSecVersion = []byte("13")
)

var (
	httpVersion10     = []byte("HTTP/1.0")
	httpVersion11     = []byte("HTTP/1.1")
	httpVersionPrefix = []byte("HTTP/")
)

type httpRequestLine struct {
	method, uri  []byte
	major, minor int
}

// httpParseRequestLine parses http request line like "GET / HTTP/1.0".
func httpParseRequestLine(line []byte) (req httpRequestLine, err error) {
	var proto []byte
	req.method, req.uri, proto = bsplit3(line, ' ')

	var ok bool
	req.major, req.minor, ok = httpParseVersion(proto)
	if !ok {
		err = ErrMalformedRequest
		return
	}

	return
}

// httpParseVersion parses major and minor version of HTTP protocol. It returns
// parsed values and true if parse is ok.
func httpParseVersion(bts []byte) (major, minor int, ok bool) {
	switch {
	case bytes.Equal(bts, httpVersion10):
		return 1, 0, true
	case bytes.Equal(bts, httpVersion11):
		return 1, 1, true
	case len(bts) < 8:
		return
	case !bytes.Equal(bts[:5], httpVersionPrefix):
		return
	}

	bts = bts[5:]

	dot := bytes.IndexByte(bts, '.')
	if dot == -1 {
		return
	}
	var err error
	major, err = asciiToInt(bts[:dot])
	if err != nil {
		return
	}
	minor, err = asciiToInt(bts[dot+1:])
	if err != nil {
		return
	}

	return major, minor, true
}

// httpParseHeaderLine parses HTTP header as key-value pair. It returns parsed
// values and true if parse is ok.
func httpParseHeaderLine(line []byte) (k, v []byte, ok bool) {
	colon := bytes.IndexByte(line, ':')
	if colon == -1 {
		return
	}

	k = btrim(line[:colon])
	// TODO(gobwas): maybe use just lower here?
	canonicalizeHeaderKey(k)

	v = btrim(line[colon+1:])
	return k, v, true
}

func btsSelectProtocol(h []byte, check func([]byte) bool) (ret string, ok bool) {
	var selected []byte
	ok = httphead.ScanTokens(h, func(v []byte) bool {
		if check(v) {
			selected = v
			return false
		}
		return true
	})
	if ok && selected != nil {
		return string(selected), true
	}
	return
}

func btsSelectExtensions(h []byte, selected []httphead.Option, check func(httphead.Option) bool) ([]httphead.Option, bool) {
	s := httphead.OptionSelector{
		Flags: httphead.SelectUnique | httphead.SelectCopy,
		Check: check,
	}
	return s.Select(h, selected)
}

func httpWriteHeader(bw *bufio.Writer, key, value string) {
	httpWriteHeaderKey(bw, key)
	_, _ = bw.WriteString(value)
	_, _ = bw.WriteString(crlf)
}

func httpWriteHeaderKey(bw *bufio.Writer, key string) {
	_, _ = bw.WriteString(key)
	_, _ = bw.WriteString(colonAndSpace)
}

func httpWriteResponseUpgrade(nonce []byte, hs Handshake, header HandshakeHeaderFunc) []byte {
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)

	_, _ = bw.WriteString(textHeadUpgrade)

	httpWriteHeaderKey(bw, headerSecAccept)
	_, _ = writeAccept(bw, nonce)
	_, _ = bw.WriteString(crlf)

	if hs.Protocol != "" {
		httpWriteHeader(bw, headerSecProtocol, hs.Protocol)
	}
	if len(hs.Extensions) > 0 {
		httpWriteHeaderKey(bw, headerSecExtensions)
		_, _ = httphead.WriteOptions(bw, hs.Extensions)
		_, _ = bw.WriteString(crlf)
	}
	if header != nil {
		_, _ = header(bw)
	}

	_, _ = bw.WriteString(crlf)

	_ = bw.Flush()
	return buf.Bytes()
}

func httpWriteResponseError(err error, code int, header HandshakeHeaderFunc) []byte {
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)

	switch code {
	case http.StatusBadRequest:
		_, _ = bw.WriteString(textHeadBadRequest)
	case http.StatusInternalServerError:
		_, _ = bw.WriteString(textHeadInternalServerError)
	case http.StatusUpgradeRequired:
		_, _ = bw.WriteString(textHeadUpgradeRequired)
	default:
		writeStatusText(bw, code)
	}

	// Write custom headers.
	if header != nil {
		_, _ = header(bw)
	}

	switch err {
	case ErrHandshakeBadProtocol:
		_, _ = bw.WriteString(textTailErrHandshakeBadProtocol)
	case ErrHandshakeBadMethod:
		_, _ = bw.WriteString(textTailErrHandshakeBadMethod)
	case ErrHandshakeBadHost:
		_, _ = bw.WriteString(textTailErrHandshakeBadHost)
	case ErrHandshakeBadUpgrade:
		_, _ = bw.WriteString(textTailErrHandshakeBadUpgrade)
	case ErrHandshakeBadConnection:
		_, _ = bw.WriteString(textTailErrHandshakeBadConnection)
	case ErrHandshakeBadSecAccept:
		_, _ = bw.WriteString(textTailErrHandshakeBadSecAccept)
	case ErrHandshakeBadSecKey:
		_, _ = bw.WriteString(textTailErrHandshakeBadSecKey)
	case ErrHandshakeBadSecVersion:
		_, _ = bw.WriteString(textTailErrHandshakeBadSecVersion)
	case ErrHandshakeUpgradeRequired:
		_, _ = bw.WriteString(textTailErrUpgradeRequired)
	case nil:
		_, _ = bw.WriteString(crlf)
	default:
		writeErrorText(bw, err)
	}

	_ = bw.Flush()
	return buf.Bytes()
}

func writeStatusText(bw *bufio.Writer, code int) {
	_, _ = bw.WriteString("HTTP/1.1 ")
	_, _ = bw.WriteString(strconv.Itoa(code))
	_ = bw.WriteByte(' ')
	_, _ = bw.WriteString(http.StatusText(code))
	_, _ = bw.WriteString(crlf)
	_, _ = bw.WriteString("Content-Type: text/plain; charset=utf-8")
	_, _ = bw.WriteString(crlf)
}

func writeErrorText(bw *bufio.Writer, err error) {
	body := err.Error()
	_, _ = bw.WriteString("Content-Length: ")
	_, _ = bw.WriteString(strconv.Itoa(len(body)))
	_, _ = bw.WriteString(crlf)
	_, _ = bw.WriteString(crlf)
	_, _ = bw.WriteString(body)
}

// statusText is a non-performant status text generator.
// NOTE: Used only to generate constants.
func statusText(code int) string {
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	writeStatusText(bw, code)
	_ = bw.Flush()
	return buf.String()
}

// errorText is a non-performant error text generator.
// NOTE: Used only to generate constants.
func errorText(err error) string {
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	writeErrorText(bw, err)
	_ = bw.Flush()
	return buf.String()
}

// HandshakeHeader is the interface that writes both upgrade request or
// response headers into a given io.Writer.
type HandshakeHeader interface {
	io.WriterTo
}

// HandshakeHeaderString is an adapter to allow the use of headers represented
// by ordinary string as HandshakeHeader.
type HandshakeHeaderString string

// WriteTo implements HandshakeHeader (and io.WriterTo) interface.
func (s HandshakeHeaderString) WriteTo(w io.Writer) (int64, error) {
	n, err := io.WriteString(w, string(s))
	return int64(n), err
}

// HandshakeHeaderBytes is an adapter to allow the use of headers represented
// by ordinary slice of bytes as HandshakeHeader.
type HandshakeHeaderBytes []byte

// WriteTo implements HandshakeHeader (and io.WriterTo) interface.
func (b HandshakeHeaderBytes) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(b)
	return int64(n), err
}

// HandshakeHeaderFunc is an adapter to allow the use of headers represented by
// ordinary function as HandshakeHeader.
type HandshakeHeaderFunc func(io.Writer) (int64, error)

// WriteTo implements HandshakeHeader (and io.WriterTo) interface.
func (f HandshakeHeaderFunc) WriteTo(w io.Writer) (int64, error) {
	return f(w)
}

// HandshakeHeaderHTTP is an adapter to allow the use of http.Header as
// HandshakeHeader.
type HandshakeHeaderHTTP http.Header

// WriteTo implements HandshakeHeader (and io.WriterTo) interface.
func (h HandshakeHeaderHTTP) WriteTo(w io.Writer) (int64, error) {
	wr := writer{w: w}
	err := http.Header(h).Write(&wr)
	return wr.n, err
}

type writer struct {
	n int64
	w io.Writer
}

func (w *writer) WriteString(s string) (int, error) {
	n, err := io.WriteString(w.w, s)
	w.n += int64(n)
	return n, err
}

func (w *writer) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.n += int64(n)
	return n, err
}
