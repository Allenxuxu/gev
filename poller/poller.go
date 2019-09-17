package poller

import "errors"

var ErrClosed = errors.New("poller instance is not running")

const waitEventsBegin = 1024

// Event represents gev configuration bit mask.
type Event uint32

// Event values
const (
	EventRead  Event = 0x1
	EventWrite Event = 0x2
	EventErr   Event = 0x80
)
