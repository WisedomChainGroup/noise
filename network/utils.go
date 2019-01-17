package network

import (
	"encoding/binary"
	"net"

)

// SerializeMessage compactly packs all bytes of a message together for cryptographic signing purposes.
func SerializeMessage(peer *Peer, message []byte) []byte {
	const uint32Size = 4

	serialized := make([]byte, uint32Size+len(peer.Address)+uint32Size+len(peer.ID())+len(message))
	pos := 0

	binary.LittleEndian.PutUint32(serialized[pos:], uint32(len(peer.Address)))
	pos += uint32Size

	copy(serialized[pos:], []byte(peer.Address))
	pos += len(peer.Address)

	binary.LittleEndian.PutUint32(serialized[pos:], uint32(len(peer.ID())))
	pos += uint32Size

	copy(serialized[pos:], peer.ID())
	pos += len(peer.ID())

	copy(serialized[pos:], message)
	pos += len(message)

	if pos != len(serialized) {
		panic("internal error: invalid serialization output")
	}

	return serialized
}


// GetRandomUnusedPort returns a random unused port
func GetRandomUnusedPort() int {
	listener, _ := net.Listen("tcp", ":0")
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
