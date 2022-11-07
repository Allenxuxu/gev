package protobuf

import "encoding/binary"

// PackMessage 按自定义协议打包数据
func PackMessage(msgType string, data []byte) []byte {
	var (
		typeLen = uint64(len(msgType))
		length  = uint64(len(data)) + typeLen + 2
		ret = make([]byte, length+4)
	)


	binary.BigEndian.PutUint32(ret, uint32(length))
	binary.BigEndian.PutUint16(ret[4:], uint16(typeLen))
	copy(ret[6:], msgType)
	copy(ret[6+typeLen:], data)

	return ret
}
