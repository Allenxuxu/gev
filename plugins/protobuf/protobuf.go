package protobuf

import "encoding/binary"

// PackMessage 按自定义协议打包数据
func PackMessage(msgType string, data []byte) []byte {
	typeLen := len(msgType)
	len := len(data) + typeLen + 2

	ret := make([]byte, len+4)

	binary.BigEndian.PutUint32(ret, uint32(len))
	binary.BigEndian.PutUint16(ret[4:], uint16(typeLen))
	copy(ret[6:], msgType)
	copy(ret[6+typeLen:], data)

	return ret
}
