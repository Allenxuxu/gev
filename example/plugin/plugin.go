package main

import (
	"github.com/Allenxuxu/gev/plugin"
	"github.com/Allenxuxu/gev/plugins/protobuf"
)

var Plugin = plugin.Config{
	Protocol: protobuf.New(),
}
