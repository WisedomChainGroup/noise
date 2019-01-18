package network

import (
	"context"
	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/log"
	"sync"
	"testing"
	"time"

	"github.com/perlin-network/noise/network/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TODO: 注释功能

var testifyTimeout = time.Second * 5

func assertNotTimeout(t *testing.T, timeout time.Duration, c chan struct{}){
	select{
	case <- c:
	case <- time.NewTimer(timeout).C:
		t.Fail()
	}
}

func newTestNetwork(port int, plugins ...PluginInterface) *Network {

	addrInfo := &AddressInfo{
		Protocol: "tcp",
		Host:     "localhost",
		Port:     uint16(port),
	}
	unifiedAddress, err := ToUnifiedAddress(addrInfo.String())
	if err != nil {
		panic(err)
	}
	network := &Network{
		opts:    defaultBuilderOptions,
		keys:    defaultBuilderOptions.signaturePolicy.RandomKeyPair(),
		Address: unifiedAddress,

		plugins:    NewPluginList(),
		transports: new(sync.Map),

		peers:       new(sync.Map),
		connections: new(sync.Map),

		listeningCh: make(chan struct{}),
		kill:        make(chan struct{}),
	}

	network.transports.Store("tcp", transport.NewTCP())
	network.transports.Store("kcp", transport.NewKCP())

	for _, p := range plugins {
		network.plugins.Put(1, p)
	}
	network.Init()
	return network
}

type NetworkTestSuite struct {
	suite.Suite
	n1 *Network
	n2 *Network
	m1 *mailBox
	m2 *mailBox
}

func(suite *NetworkTestSuite) SetupSuite(){
	log.Disable()
}

func(suite *NetworkTestSuite) SetupTest(){
	suite.m1 = newMailBox()
	suite.n1 = newTestNetwork(6002, suite.m1)
	suite.m2 = newMailBox()
	suite.n2 = newTestNetwork(6003, suite.m2)
	go suite.n1.Listen()
	go suite.n2.Listen()
	suite.n1.BlockUntilListening()
	suite.n2.BlockUntilListening()
}

func(suite *NetworkTestSuite) TearDownTest(){
	suite.n1.Close()
	suite.n2.Close()
	<- suite.n1.kill
	<- suite.n2.kill
}


func (suite *NetworkTestSuite) TestListen() {
	t := suite.T()
	m := newMailBox()
	n := newTestNetwork(6000, m)
	go n.Listen()
	assertNotTimeout(t, testifyTimeout, m.started)
	n.Close()
}

func (suite *NetworkTestSuite) TestClose(){
	t := suite.T()
	m := newMailBox()
	n := newTestNetwork(6000, m)
	go n.Listen()
	n.Close()
	assertNotTimeout(t, testifyTimeout, m.closed)
}

func (suite *NetworkTestSuite) TestBootstrap() {
	t := suite.T()
	suite.n1.Bootstrap(suite.n2.Self().Encode())
	assertNotTimeout(t, testifyTimeout, suite.m1.peerConnected)
	assertNotTimeout(t, testifyTimeout, suite.m2.peerConnected)
	_, ok := suite.n1.connections.Load(suite.n2.Self().ID())
	assert.True(t, ok)
	_, ok = suite.n1.peers.Load(suite.n2.Self().ID())
	assert.True(t, ok)
	_, ok = suite.n2.connections.Load(suite.n1.Self().ID())
	assert.True(t, ok)
	_, ok = suite.n2.peers.Load(suite.n1.Self().ID())
	assert.True(t, ok)
}

func(suite *NetworkTestSuite) TestBroadcast(){
	t := suite.T()
	suite.n1.Bootstrap(suite.n2.Self().Encode())
	suite.n1.Broadcast(WithSignMessage(context.Background(), true), &protobuf.Ping{})
	ping := <- suite.m2.messages
	_ , ok := ping.(*protobuf.Ping)
	assert.True(t, ok)
}

func(suite *NetworkTestSuite) TestBroadcastByID(){
	t := suite.T()
	suite.n1.Bootstrap(suite.n2.Self().Encode())
	suite.n1.BroadcastByIDs(
		WithSignMessage(context.Background(), true), &protobuf.Ping{}, suite.n2.Self().ID(),
		)
	ping := <- suite.m2.messages
	_ , ok := ping.(*protobuf.Ping)
	assert.True(t, ok)
}

func(suite *NetworkTestSuite) TestPeerDisconnected(){
	t := suite.T()
	suite.n1.Bootstrap(suite.n2.Self().Encode())
	// block until peers join
	<- suite.m2.peerConnected
	conn, ok := suite.n2.connections.Load(suite.n1.Self().ID())
	assert.True(t, ok)
	c, ok := conn.(*ConnState)
	assert.True(t, ok)
	assert.NoError(t,c.conn.Close())
	assertNotTimeout(t, testifyTimeout, suite.m1.peerDisconnected)
	assertNotTimeout(t, testifyTimeout, suite.m2.peerDisconnected)
	_, ok = suite.n1.connections.Load(suite.n2.ID())
	assert.False(t, ok)
	_, ok = suite.n1.peers.Load(suite.n2.ID())
	assert.False(t, ok)
}

func TestNetworkTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkTestSuite))
}
