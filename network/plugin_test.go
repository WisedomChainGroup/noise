package network

import (
	"sync/atomic"
)

// plugin for network test
type mailBox struct {
	Plugin
	messages         chan interface{}
	contexts         chan interface{}
	started          chan interface{}
	closed           chan interface{}
	peerConnected    chan interface{}
	peerDisconnected chan interface{}
}

func newMailBox() *mailBox {
	return &mailBox{
		messages:         make(chan interface{}, 1),
		started:          make(chan interface{}),
		closed:           make(chan interface{}),
		peerConnected:    make(chan interface{}, 1),
		peerDisconnected: make(chan interface{}, 1),
		contexts:         make(chan interface{}, 1),
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
	m.peerConnected <- struct{}{}
}

func (m *mailBox) PeerDisconnect(*PeerClient) {
	m.peerDisconnected <- struct{}{}
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
