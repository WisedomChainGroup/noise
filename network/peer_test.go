package network

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PeerTestSuite struct {
	suite.Suite
}

func newTestPeer() *Peer {
	pbk := []byte{'a', 'b', 'c'}
	address := "localhost:9090"
	return &Peer{
		PublicKey: pbk,
		Address:   address,
	}
}

// TestEncode 测试节点编码
func (suite *PeerTestSuite) TestEncode() {
	t := suite.T()
	peer := newTestPeer()
	assert.Equal(t, fmt.Sprintf("%s://%s@%s", protocolName, hex.EncodeToString(peer.PublicKey), peer.Address), peer.Encode())
}

// TestDecode 测试节点解码
func(suite *PeerTestSuite) TestDecode(){
	t := suite.T()
	peer := newTestPeer()
	peer2, err := Decode(fmt.Sprintf("%s://%s@%s", protocolName, hex.EncodeToString([]byte{'a', 'b', 'c'}),"localhost:9090"))
	assert.NoError(t, err)
	assert.Equal(t, peer.Encode(), peer2.Encode())
}

// TestPeerTestSuite 启动测试
func TestPeerTestSuite(t *testing.T) {
	suite.Run(t, new(PeerTestSuite))
}
