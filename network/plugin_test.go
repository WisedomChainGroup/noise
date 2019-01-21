package network

import (
	"sync/atomic"

	"github.com/gogo/protobuf/proto"
)

// plugin for network test
type mailBox struct {
	Plugin
	messages         chan proto.Message
	contexts         chan *PluginContext
	started          chan struct{}
	closed           chan struct{}
	peerConnected    chan struct{}
	peerDisconnected chan struct{}
}

func newMailBox() *mailBox {
	return &mailBox{
		messages:         make(chan proto.Message, 1),
		started:          make(chan struct{}),
		closed:           make(chan struct{}),
		peerConnected:    make(chan struct{}),
		peerDisconnected: make(chan struct{}),
		contexts:         make(chan *PluginContext),
	}
}

func (m *mailBox) Startup(*Network) {
	close(m.started)
}

func (m *mailBox) Receive(ctx *PluginContext) error {
	m.messages <- ctx.message
	m.contexts <- ctx
	return nil
}

func (m *mailBox) Cleanup(*Network) {
	close(m.closed)
}

func (m *mailBox) PeerConnect(*PeerClient) {
	close(m.peerConnected)
}

func (m *mailBox) PeerDisconnect(*PeerClient) {
	close(m.peerDisconnected)
}

type MockPlugin struct {
	*Plugin

	startup        int32
	receive        int32
	cleanup        int32
	peerConnect    int32
	peerDisconnect int32
}

func (p *MockPlugin) Startup(net *Network) {
	atomic.AddInt32(&p.startup, 1)
}

func (p *MockPlugin) Receive(ctx *PluginContext) error {
	atomic.AddInt32(&p.receive, 1)
	return nil
}

func (p *MockPlugin) Cleanup(net *Network) {
	atomic.AddInt32(&p.cleanup, 1)
}

func (p *MockPlugin) PeerConnect(client *PeerClient) {
	atomic.AddInt32(&p.peerConnect, 1)
}

func (p *MockPlugin) PeerDisconnect(client *PeerClient) {
	atomic.AddInt32(&p.peerDisconnect, 1)
}
