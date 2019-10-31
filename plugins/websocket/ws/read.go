package ws

import (
	"encoding/binary"
	"fmt"

	"github.com/Allenxuxu/ringbuffer"
	"github.com/Allenxuxu/toolkit/convert"
	"github.com/gobwas/pool/pbytes"
)

// Errors used by frame reader.
var (
	ErrHeaderLengthMSB        = fmt.Errorf("header error: the most significant bit must be 0")
	ErrHeaderLengthUnexpected = fmt.Errorf("header error: unexpected payload length bits")
	ErrHeaderNotReady         = fmt.Errorf("header error: not enough")
)

// VirtualReadHeader reads a frame header from r.
func VirtualReadHeader(in *ringbuffer.RingBuffer) (h Header, err error) {
	if in.Length() < 6 {
		err = ErrHeaderNotReady
		return
	}

	bts := pbytes.Get(2, MaxHeaderSize-2)
	defer pbytes.Put(bts)
	// Prepare to hold first 2 bytes to choose size of next read.
	_, _ = in.VirtualRead(bts)

	h.Fin = bts[0]&bit0 != 0
	h.Rsv = (bts[0] & 0x70) >> 4
	h.OpCode = OpCode(bts[0] & 0x0f)

	var extra int

	if bts[1]&bit0 != 0 {
		h.Masked = true
		extra += 4
	}

	length := bts[1] & 0x7f
	switch {
	case length < 126:
		h.Length = int64(length)

	case length == 126:
		extra += 2

	case length == 127:
		extra += 8

	default:
		err = ErrHeaderLengthUnexpected
		return
	}

	if extra == 0 {
		return
	}

	// Increase len of bts to extra bytes need to read.
	// Overwrite first 2 bytes that was read before.
	bts = bts[:extra]
	_, _ = in.VirtualRead(bts)

	switch {
	case length == 126:
		h.Length = int64(binary.BigEndian.Uint16(bts[:2]))
		bts = bts[2:]

	case length == 127:
		if bts[0]&0x80 != 0 {
			err = ErrHeaderLengthMSB
			return
		}
		h.Length = int64(binary.BigEndian.Uint64(bts[:8]))
		bts = bts[8:]
	}

	if h.Masked {
		copy(h.Mask[:], bts)
	}

	return
}

// ParseCloseFrameData parses close frame status code and closure reason if any provided.
// If there is no status code in the payload
// the empty status code is returned (code.Empty()) with empty string as a reason.
func ParseCloseFrameData(payload []byte) (code StatusCode, reason string) {
	if len(payload) < 2 {
		// We returning empty StatusCode here, preventing the situation
		// when endpoint really sent code 1005 and we should return ProtocolError on that.
		//
		// In other words, we ignoring this rule [RFC6455:7.1.5]:
		//   If this Close control frame contains no status code, _The WebSocket
		//   Connection Close Code_ is considered to be 1005.
		return
	}
	code = StatusCode(binary.BigEndian.Uint16(payload))
	reason = convert.BytesToString(payload[2:])
	return
}
