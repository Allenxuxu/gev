package poller

// Event represents gev configuration bit mask.
type Event uint32

// Event values
const (
	EventRead  Event = 0x1
	EventWrite       = 0x2
	EventErr         = 0x80
)
