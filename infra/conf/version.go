package conf

import (
	"github.com/mrst2000/Xray-core/app/version"
	"github.com/mrst2000/Xray-core/core"
	"strconv"
)

type VersionConfig struct {
	MinVersion string `json:"min"`
	MaxVersion string `json:"max"`
}

func (c *VersionConfig) Build() (*version.Config, error) {
	coreVersion := strconv.Itoa(int(core.Version_x)) + "." + strconv.Itoa(int(core.Version_y)) + "." + strconv.Itoa(int(core.Version_z))

	return &version.Config{
		CoreVersion: coreVersion,
		MinVersion:  c.MinVersion,
		MaxVersion:  c.MaxVersion,
	}, nil
}
