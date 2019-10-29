package main

import (
	"encoding/binary"
	"github.com/Allenxuxu/ringbuffer"
	"github.com/gobwas/pool/pbytes"
)

const exampleHeaderLen = 4

type ExampleDataPack struct{}

func (d *ExampleDataPack) UnPacket(buffer *ringbuffer.RingBuffer) []byte {
	if buffer.VirtualLength() > exampleHeaderLen {
		buf := pbytes.GetLen(exampleHeaderLen)
		defer pbytes.Put(buf)
		_, _ = buffer.VirtualRead(buf)
		dataLen := binary.BigEndian.Uint32(buf)

		if buffer.VirtualLength() >= int(dataLen) {
			ret := make([]byte, dataLen)
			_, _ = buffer.VirtualRead(ret)

			buffer.VirtualFlush()
			return ret
		} else {
			buffer.VirtualRevert()
		}
	}
	return nil
}

func (d *ExampleDataPack) Packet(data []byte) []byte {
	dataLen := len(data)
	ret := make([]byte, exampleHeaderLen+dataLen)
	binary.BigEndian.PutUint32(ret, uint32(dataLen))
	copy(ret[4:], data)
	return ret
}
