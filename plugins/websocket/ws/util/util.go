package util

import (
	"unicode/utf8"

	"github.com/Allenxuxu/gev/plugins/websocket/ws"
)

// PackData 封装 websocket message 数据包
func PackData(messageType ws.MessageType, data []byte) ([]byte, error) {
	var frame *ws.Frame
	switch messageType {
	case ws.MessageBinary:
		frame = ws.NewBinaryFrame(data)
	case ws.MessageText:
		frame = ws.NewTextFrame(data)
	}
	return ws.FrameToBytes(frame)
}

// PackCloseData 封装 websocket close 数据包
func PackCloseData(reason string) ([]byte, error) {
	return ws.FrameToBytes(ws.NewCloseFrame(ws.NewCloseFrameBody(ws.StatusNormalClosure, reason)))
}

// HandleClose 处理 websocket close 返回应答信息
func HandleClose(h *ws.Header, payload []byte) ([]byte, error) {
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

// HandlePing 处理 ping 返回应答信息
func HandlePing(payload []byte) ([]byte, error) {
	return ws.FrameToBytes(ws.NewPongFrame(payload))
}

// HandlePong 处理 pong 返回应答信息
func HandlePong(payload []byte) ([]byte, error) {
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
