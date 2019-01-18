package network

import (
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BuilderTestSuite struct {
	suite.Suite
}

// TestNewBuilder 测试创建 builder
func (suite *BuilderTestSuite) TestNewBuilder() {
	t := suite.T()
	builder := NewBuilder()
	assert.NotNil(t, builder)
	assert.Equal(t, defaultAddress, builder.address)
	assert.Equal(t, defaultConnectionTimeout, builder.opts.connectionTimeout)
	_, ok := builder.opts.signaturePolicy.(*ed25519.Ed25519)
	assert.True(t, ok)
	_, ok = builder.opts.hashPolicy.(*blake2b.Blake2b)
	assert.True(t, ok)
	assert.Equal(t, defaultReceiveWindowSize, builder.opts.recvWindowSize)
	assert.Equal(t, defaultSendWindowSize, builder.opts.sendWindowSize)
	assert.Equal(t, defaultWriteBufferSize, builder.opts.writeBufferSize)
	assert.Equal(t, defaultWriteFlushLatency, builder.opts.writeFlushLatency)
	assert.Equal(t, defaultWriteTimeout, builder.opts.writeTimeout)
	assert.Equal(t, defaultAddress, builder.address)
}

// TestBuildNetwork 测试 build 出来的 network
func (suite *BuilderTestSuite) TestBuildNetwork() {
	t := suite.T()
	builder := NewBuilder()
	mailbox := newMailBox()
	assert.NoError(t, builder.AddPlugin(mailbox))
	network, err := builder.Build()
	assert.NoError(t, err)
	assert.Equal(t, defaultConnectionTimeout, network.opts.connectionTimeout)
	assert.NotZero(t, network.plugins.Len())
}



func TestBuilderTestSuite(t *testing.T) {
	suite.Run(t, new(BuilderTestSuite))
}
