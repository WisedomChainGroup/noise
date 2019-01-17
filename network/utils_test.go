package network

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestSerializeMessageInfoForSigning(t *testing.T) {
	mustReadRand := func(size int) []byte {
		out := make([]byte, size)
		_, err := rand.Read(out)
		if err != nil {
			panic(err)
		}
		return out
	}

	pk1, pk2 := mustReadRand(32), mustReadRand(32)

	peers := []*Peer{
		&Peer{Address: "tcp://127.0.0.1:3001", PublicKey: pk1},
		&Peer{Address: "tcp://127.0.0.1:3001", PublicKey: pk2},
		&Peer{Address: "tcp://127.0.0.1:3002", PublicKey: pk1},
		&Peer{Address: "tcp://127.0.0.1:3002", PublicKey: pk2},
	}

	messages := [][]byte{
		[]byte("hello"),
		[]byte("world"),
	}

	outputs := make([][]byte, 0)

	for _, p := range peers {
		for _, msg := range messages {
			outputs = append(outputs, SerializeMessage(p, msg))
		}
	}

	for i := 0; i < len(outputs); i++ {
		for j := i + 1; j < len(outputs); j++ {
			if bytes.Equal(outputs[i], outputs[j]) {
				t.Fatal("Different inputs produced the same output")
			}
		}
	}
}

