// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tcp

import (
	"sync"

	"github.com/liuxp0827/net-protocol/pkg/buffer"
	"github.com/liuxp0827/net-protocol/pkg/seqnum"
	"github.com/liuxp0827/net-protocol/pkg/waiter"
	tcpip "github.com/liuxp0827/net-protocol/protocol"
	"github.com/liuxp0827/net-protocol/protocol/header"
	"github.com/liuxp0827/net-protocol/stack"
)

// Forwarder is a connection request forwarder, which allows clients to decide
// what to do with a connection request, for example: ignore it, send a RST, or
// attempt to complete the 3-way handshake.
//
// The canonical way of using it is to pass the Forwarder.HandlePacket function
// to stack.SetTransportProtocolHandler.
// Forwarder是一个连接请求转发器，它允许客户端决定如何处理连接请求，例如：忽略它，发送RST或尝试完成3次握手。
//
// 使用它的规范方法是将Forwarder.HandlePacket函数传递给stack.SetTransportProtocolHandler。
type Forwarder struct {
	maxInFlight int
	handler     func(*ForwarderRequest)

	mu       sync.Mutex
	inFlight map[stack.TransportEndpointID]struct{}
	listen   *listenContext
}

// NewForwarder allocates and initializes a new forwarder with the given
// maximum number of in-flight connection attempts. Once the maximum is reached
// new incoming connection requests will be ignored.
//
// If rcvWnd is set to zero, the default buffer size is used instead.
func NewForwarder(s *stack.Stack, rcvWnd, maxInFlight int, handler func(*ForwarderRequest)) *Forwarder {
	if rcvWnd == 0 {
		rcvWnd = DefaultBufferSize
	}
	return &Forwarder{
		maxInFlight: maxInFlight,
		handler:     handler,
		inFlight:    make(map[stack.TransportEndpointID]struct{}),
		listen:      newListenContext(s, seqnum.Size(rcvWnd), true, 0),
	}
}

// HandlePacket handles a packet if it is of interest to the forwarder (i.e., if
// it's a SYN packet), returning true if it's the case. Otherwise the packet
// is not handled and false is returned.
//
// This function is expected to be passed as an argument to the
// stack.SetTransportProtocolHandler function.
func (f *Forwarder) HandlePacket(r *stack.Route, id stack.TransportEndpointID, vv buffer.VectorisedView) bool {
	s := newSegment(r, id, vv)
	defer s.decRef()

	// We only care about well-formed SYN packets.
	if !s.parse() || s.flags != flagSyn {
		return false
	}

	opts := parseSynSegmentOptions(s)

	f.mu.Lock()
	defer f.mu.Unlock()

	// We have an inflight request for this id, ignore this one for now.
	if _, ok := f.inFlight[id]; ok {
		return true
	}

	// Ignore the segment if we're beyond the limit.
	if len(f.inFlight) >= f.maxInFlight {
		return true
	}

	// Launch a new goroutine to handle the request.
	f.inFlight[id] = struct{}{}
	s.incRef()
	go f.handler(&ForwarderRequest{
		forwarder:  f,
		segment:    s,
		synOptions: opts,
	})

	return true
}

// ForwarderRequest represents a connection request received by the forwarder
// and passed to the client. Clients must eventually call Complete() on it, and
// may optionally create an endpoint to represent it via CreateEndpoint.
type ForwarderRequest struct {
	mu         sync.Mutex
	forwarder  *Forwarder
	segment    *segment
	synOptions header.TCPSynOptions
}

// ID returns the 4-tuple (src address, src port, dst address, dst port) that
// represents the connection request.
func (r *ForwarderRequest) ID() stack.TransportEndpointID {
	return r.segment.id
}

// Complete completes the request, and optionally sends a RST segment back to the
// sender.
func (r *ForwarderRequest) Complete(sendReset bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.segment == nil {
		panic("Completing already completed forwarder request")
	}

	// Remove request from the forwarder.
	r.forwarder.mu.Lock()
	delete(r.forwarder.inFlight, r.segment.id)
	r.forwarder.mu.Unlock()

	// If the caller requested, send a reset.
	if sendReset {
		replyWithReset(r.segment)
	}

	// Release all resources.
	r.segment.decRef()
	r.segment = nil
	r.forwarder = nil
}

// CreateEndpoint creates a TCP endpoint for the connection request, performing
// the 3-way handshake in the process.
func (r *ForwarderRequest) CreateEndpoint(queue *waiter.Queue) (tcpip.Endpoint, *tcpip.Error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.segment == nil {
		return nil, tcpip.ErrInvalidEndpointState
	}

	f := r.forwarder
	ep, err := f.listen.createEndpointAndPerformHandshake(r.segment, &header.TCPSynOptions{
		MSS:           r.synOptions.MSS,
		WS:            r.synOptions.WS,
		TS:            r.synOptions.TS,
		TSVal:         r.synOptions.TSVal,
		TSEcr:         r.synOptions.TSEcr,
		SACKPermitted: r.synOptions.SACKPermitted,
	})
	if err != nil {
		return nil, err
	}

	// Start the protocol goroutine.
	ep.startAcceptedLoop(queue)

	return ep, nil
}
