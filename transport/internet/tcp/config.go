package tcp

import (
	"github.com/mrst2000/Xray-core/common"
	"github.com/mrst2000/Xray-core/transport/internet"
)

func init() {
	common.Must(internet.RegisterProtocolConfigCreator(protocolName, func() interface{} {
		return new(Config)
	}))
}
