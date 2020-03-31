package plugin

import (
	"errors"
	pg "plugin"

	"github.com/Allenxuxu/gev/connection"
)

// Config is the plugin config
type Config struct {
	Protocol connection.Protocol
}

func Load(path string) (*Config, error) {
	plugin, err := pg.Open(path)
	if err != nil {
		return nil, err
	}
	s, err := plugin.Lookup("Plugin")
	if err != nil {
		return nil, err
	}
	pl, ok := s.(*Config)
	if !ok {
		return nil, errors.New("could not cast Plugin object")
	}
	return pl, nil
}
