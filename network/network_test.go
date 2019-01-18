package network

import (
	"sync"
	"testing"
	"time"

	"github.com/perlin-network/noise/network/transport"
	"github.com/stretchr/testify/suite"
)

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

//func (suite *NetworkTestSuite) TestBootstrap() {
//	m1 := newMailBox()
//	n1 := newTestNetwork(6002, m1)
//	m2 := newMailBox()
//	n2 := newTestNetwork(6003, m2)
//	go n1.Listen()
//	go n2.Listen()
//	n1.Bootstrap(n2.Self().Encode())
//	// block until peer connected
//	<-m1.peerConnected
//	n1.Close()
//	n2.Close()
//}

func TestNetworkTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkTestSuite))
}
