package datapacket

import (
	"github.com/Allenxuxu/ringbuffer"
)

type DataPacket interface {
	UnPacket(buffer *ringbuffer.RingBuffer) []byte
	Packet(data []byte) []byte
}

type DefaultDataPack struct{}

func (d *DefaultDataPack) UnPacket(buffer *ringbuffer.RingBuffer) []byte {
	ret := buffer.Bytes()
	buffer.RetrieveAll()
	return ret
}

func (d *DefaultDataPack) Packet(data []byte) []byte {
	return data
}
