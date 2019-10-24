package handler

import (
	"log"

	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/ws"
	"github.com/Allenxuxu/ringbuffer"
	"unicode/utf8"
)

// HandleWebSocket 处理 WebSocket 信息
func HandleWebSocket(c *connection.Connection, buffer *ringbuffer.RingBuffer,
	handler func(*connection.Connection, []byte) (ws.MessageType, []byte)) (out []byte) {

	header, err := ws.VirtualReadHeader(buffer)
	if err != nil {
		log.Println(err)
		return
	}
	if buffer.Length() >= int(header.Length) {
		buffer.VirtualFlush()

		payload := make([]byte, int(header.Length))
		_, _ = buffer.Read(payload)

		if header.Masked {
			ws.Cipher(payload, header.Mask, 0)
		}

		if header.OpCode.IsControl() {
			switch header.OpCode {
			case ws.OpClose:
				out, err = handlerClose(&header, payload)
				if err != nil {
					log.Println(err)
				}
				_ = c.ShutdownWrite()
			case ws.OpPing:
				out, err = handlerPing(payload)
				if err != nil {
					log.Println(err)
				}
			case ws.OpPong:
				out, err = handlerPong(payload)
				if err != nil {
					log.Println(err)
				}
			}
			return
		}

		messageType, data := handler(c, payload)
		if len(data) > 0 {
			var frame *ws.Frame
			switch messageType {
			case ws.MessageBinary:
				frame = ws.NewBinaryFrame(data)
			case ws.MessageText:
				frame = ws.NewTextFrame(data)
			}
			out, err = ws.FrameToBytes(frame)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}

	return
}

func handlerClose(h *ws.Header, payload []byte) ([]byte, error) {
	if h.Length == 0 {
		return ws.WriteHeader(&ws.Header{
			Fin:    true,
			OpCode: ws.OpClose,
		})
	}

	code, reason := ws.ParseCloseFrameData(payload)
	if err := CheckCloseFrameData(code, reason); err != nil {
		// Here we could not use the prepared bytes because there is no
		// guarantee that it may fit our protocol error closure code and a
		// reason.
		return ws.FrameToBytes(ws.NewCloseFrame(ws.NewCloseFrameBody(
			ws.StatusProtocolError, err.Error(),
		)))
	}

	return ws.FrameToBytes(ws.NewCloseFrame(ws.NewCloseFrameBody(code, reason)))
}

func handlerPing(payload []byte) ([]byte, error) {
	return ws.FrameToBytes(ws.NewPongFrame(payload))
}

func handlerPong(payload []byte) ([]byte, error) {
	return ws.FrameToBytes(ws.NewPingFrame(payload))
}

// CheckCloseFrameData checks received close information
// to be valid RFC6455 compatible close info.
//
// Note that code.Empty() or code.IsAppLevel() will raise error.
//
// If endpoint sends close frame without status code (with frame.Length = 0),
// application should not check its payload.
func CheckCloseFrameData(code ws.StatusCode, reason string) error {
	switch {
	case code.IsNotUsed():
		return ws.ErrProtocolStatusCodeNotInUse

	case code.IsProtocolReserved():
		return ws.ErrProtocolStatusCodeApplicationLevel

	case code == ws.StatusNoMeaningYet:
		return ws.ErrProtocolStatusCodeNoMeaning

	case code.IsProtocolSpec() && !code.IsProtocolDefined():
		return ws.ErrProtocolStatusCodeUnknown

	case !utf8.ValidString(reason):
		return ws.ErrProtocolInvalidUTF8

	default:
		return nil
	}
}
