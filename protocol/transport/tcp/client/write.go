package client

import (
	"github.com/liuxp0827/net-protocol/pkg/buffer"
	tcpip "github.com/liuxp0827/net-protocol/protocol"
)

// Write
func (c *Client) Write(buf []byte) error {
	v := buffer.View(buf)
	c.ep.Write(tcpip.SlicePayload(v),
		tcpip.WriteOptions{To: &c.remote})
	return nil
}
