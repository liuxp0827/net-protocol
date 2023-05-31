package dns

import "github.com/liuxp0827/net-protocol/protocol/header"

//sendQuery udp query dns
func (e *Endpoint) sendQuery() (*[]header.DNSResource, error) {

	if err := e.c.Connect(); err != nil {
		return nil, err
	}
	if err := e.c.Write(*e.req); err != nil {
		return nil, err
	}
	return e.parseResp()
}