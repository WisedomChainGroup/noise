package network

import (
	"encoding/hex"
	"fmt"
	"github.com/perlin-network/noise/internal/protobuf"
	"net/url"
)

type Peer protobuf.Peer

const protocolName = "wnode"

// Encode return url
func(p *Peer) Encode() string{
	return fmt.Sprintf("%s://%s@%s", protocolName, p.ID(), p.Address)
}

func(p *Peer) ID() string{
	return hex.EncodeToString(p.PublicKey)
}

func(p *Peer) Equal(p2 *Peer) bool{
	return p.ID() == p2.ID()
}

// Decode url
func Decode(raw string) (*Peer, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	pubkey, err := hex.DecodeString(u.User.String())
	if err != nil {
		return nil, err
	}
	return &Peer{
		Address:   u.Host,
		PublicKey: pubkey,
	}, nil
}
