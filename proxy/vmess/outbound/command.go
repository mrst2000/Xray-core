package outbound

import (

	"github.com/mrst2000/Xray-core/common/net"
	"github.com/mrst2000/Xray-core/common/protocol"
)

// As a stub command consumer.
func (h *Handler) handleCommand(dest net.Destination, cmd protocol.ResponseCommand) {
	switch cmd.(type) {
	default:
	}
}
