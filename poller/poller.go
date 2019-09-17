package poller

// Event represents gev configuration bit mask.
type Event uint32

// Event values
const (
	EventRead  Event = 0x1
	EventWrite Event = 0x2
	EventErr   Event = 0x80
)
