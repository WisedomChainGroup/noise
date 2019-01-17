package network

import (
	"bufio"
	"context"
	"encoding/hex"
	"math/rand"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/network/transport"
	"github.com/perlin-network/noise/types/opcode"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

const (
	defaultConnectionTimeout = 60 * time.Second
	defaultReceiveWindowSize = 4096
	defaultSendWindowSize    = 4096
	defaultWriteBufferSize   = 4096
	defaultWriteFlushLatency = 50 * time.Millisecond
	defaultWriteTimeout      = 3 * time.Second
)

// TODO: 完善单元测试 路由插件

var contextPool = sync.Pool{
	New: func() interface{} {
		return new(PluginContext)
	},
}

var (
	_ NetworkInterface = (*Network)(nil)
)

// Network represents the current networking state for this node.
type Network struct {
	opts options

	// Node's keypair.
	keys *crypto.KeyPair

	// Full address to listen on. `protocol://host:port`
	Address string

	// Map of plugins registered to the network.
	// map[string]Plugin
	plugins *PluginList

	// Map of connection addresses (string) <-> *network.PeerClient
	// so that the Network doesn't dial multiple times to the same ip
	peers *sync.Map

	//RecvQueue chan *protobuf.Message

	// Map of connection addresses (string) <-> *ConnState
	connections *sync.Map

	// Map of protocol addresses (string) <-> *transport.Layer
	transports *sync.Map

	// listeningCh will block a goroutine until this node is listening for peers.
	listeningCh chan struct{}

	// <-kill will begin the server shutdown process
	kill chan struct{}
}

// options for network struct
type options struct {
	connectionTimeout time.Duration
	signaturePolicy   crypto.SignaturePolicy
	hashPolicy        crypto.HashPolicy
	recvWindowSize    int
	sendWindowSize    int
	writeBufferSize   int
	writeFlushLatency time.Duration
	writeTimeout      time.Duration
}

// ConnState represents a connection.
type ConnState struct {
	conn         net.Conn
	writer       *bufio.Writer
	messageNonce uint64
	writerMutex  *sync.Mutex
}

// Init starts all network I/O workers.
func (n *Network) Init() {
	// Spawn write flusher.
	go n.flushLoop()
}

func(n *Network) ID() string{
	return hex.EncodeToString(n.keys.PublicKey)
}

func(n *Network) Self() *Peer{
	addrInfo, err := ParseAddress(n.Address)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	return &Peer{
		PublicKey:n.keys.PublicKey,
		Address: addrInfo.HostPort(),
	}
}

func (n *Network) flushLoop() {
	t := time.NewTicker(n.opts.writeFlushLatency)
	defer t.Stop()
	for {
		select {
		case <-n.kill:
			return
		case <-t.C:
			n.connections.Range(func(key, value interface{}) bool {
				if state, ok := value.(*ConnState); ok {
					state.writerMutex.Lock()
					if err := state.writer.Flush(); err != nil {
						log.Warn().Err(err).Msg("")
					}
					state.writerMutex.Unlock()
				}
				return true
			})
		}
	}
}

// GetKeys returns the keypair for this network
func (n *Network) GetKeys() *crypto.KeyPair {
	return n.keys
}

func (n *Network) dispatchMessage(client *PeerClient, msg *protobuf.Message) {
	var ptr proto.Message
	// unmarshal message based on specified opcode
	code := opcode.Opcode(msg.Opcode)
	switch code {
	case opcode.BytesCode:
		ptr = new(protobuf.Bytes)
	case opcode.PingCode:
		ptr = new(protobuf.Ping)
	case opcode.PongCode:
		ptr = new(protobuf.Pong)
	case opcode.LookupNodeRequestCode:
		ptr = new(protobuf.LookupNodeRequest)
	case opcode.LookupNodeResponseCode:
		ptr = new(protobuf.LookupNodeResponse)
	case opcode.UnregisteredCode:
		log.Error().Msg("network: message received had no opcode")
		return
	default:
		var err error
		ptr, err = opcode.GetMessageType(code)
		if err != nil {
			log.Error().Err(err).Msg("network: received message opcode is not registered")
			return
		}
	}

	if len(msg.Message) > 0 {
		if err := proto.Unmarshal(msg.Message, ptr); err != nil {
			log.Error().Msgf("%v", err)
			return
		}
	}

	if msg.RequestNonce > 0 && msg.ReplyFlag {
		if _state, exists := client.Requests.Load(msg.RequestNonce); exists {
			state := _state.(*RequestState)
			select {
			case state.data <- ptr:
			case <-state.closeSignal:
			}
			return
		}
	}

	switch msgRaw := ptr.(type) {
	case *protobuf.Bytes:
		client.handleBytes(msgRaw.Data)
	default:
		ctx := contextPool.Get().(*PluginContext)
		ctx.client = client
		ctx.message = msgRaw
		ctx.nonce = msg.RequestNonce

		go func() {
			// Execute 'on receive message' callback for all plugins.
			n.plugins.Each(func(plugin PluginInterface) {
				if err := plugin.Receive(ctx); err != nil {
					log.Error().Err(err).Msg("")
				}
			})

			contextPool.Put(ctx)
		}()
	}
}

// Listen starts listening for peers on a port.
func (n *Network) Listen() {
	// Handle 'network starts listening' callback for plugins.
	n.plugins.Each(func(plugin PluginInterface) {
		plugin.Startup(n)
	})

	// Handle 'network stops listening' callback for plugins.
	defer func() {
		n.plugins.Each(func(plugin PluginInterface) {
			plugin.Cleanup(n)
		})
	}()

	addrInfo, err := ParseAddress(n.Address)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	var listener net.Listener

	if t, exists := n.transports.Load(addrInfo.Protocol); exists {
		listener, err = t.(transport.Layer).Listen(int(addrInfo.Port))
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
	} else {
		err := errors.New("network: invalid protocol " + addrInfo.Protocol)
		log.Fatal().Err(err).Msg("")
	}

	n.startListening()

	log.Info().
		Str("address", n.Self().Encode()).
		Msg("Listening for peers.")

	// handle server shutdowns
	go func() {
		select {
		case <-n.kill:
			// cause listener.Accept() to stop blocking so it can continue the loop
			listener.Close()
		}
	}()

	// Handle new clients.
	for {
		if conn, err := listener.Accept(); err == nil {
			go n.Accept(conn)
		} else {
			// if the Shutdown flag is set, no need to continue with the for loop
			select {
			case <-n.kill:
				log.Info().Msgf("Shutting down server %s.", n.Address)
				return
			default:
				log.Error().Msgf("%v", err)
			}
		}
	}
}

// Client either creates or returns a cached peer client given its host address.
func (n *Network) Client(peer *Peer, conn net.Conn) (*PeerClient, error) {

	clientNew := createPeerClient(n, peer.ID(), conn)


	c, exists := n.peers.LoadOrStore(peer.ID(), clientNew)
	if exists {
		client := c.(*PeerClient)
		return client, nil
	}

	client := c.(*PeerClient)

	n.connections.Store(peer.ID(), &ConnState{
		conn:        conn,
		writer:      bufio.NewWriterSize(conn, n.opts.writeBufferSize),
		writerMutex: new(sync.Mutex),
	})

	client.Init()

	go n.receiveLoop(client, conn)

	return client, nil
}

// ConnectionStateExists returns true if network has a connection on a given peer id.
func (n *Network) ConnectionStateExists(id string) bool {
	_, ok := n.connections.Load(id)
	return ok
}

// ConnectionState returns a connections state for current address.
func (n *Network) ConnectionState(id string) (*ConnState, bool) {
	conn, ok := n.connections.Load(id)
	if !ok {
		return nil, false
	}
	return conn.(*ConnState), true
}

// startListening will start node for listening for new peers.
func (n *Network) startListening() {
	close(n.listeningCh)
}

// BlockUntilListening blocks until this node is listening for new peers.
func (n *Network) BlockUntilListening() {
	<-n.listeningCh
}

// Bootstrap with a number of encoded peers
func (n *Network) Bootstrap(raws ...string) {
	n.BlockUntilListening()

	for _, r := range raws{
		p, err := Decode(r)
		if err != nil{
			log.Error().Err(err).Msg("")
			continue
		}
		err = n.AddPeer(p)
		if err != nil{
			log.Error().Err(err).Msg("")
			continue
		}
	}
}

func (n *Network) AddPeer(p *Peer) error{
	conn, err := n.Dial(p.Address)
	if err != nil{
		return err
	}
	_, err = n.Client(p, conn)
	return err
}

// Dial establishes a bidirectional connection to an address, and additionally handshakes with said address.
func (n *Network) Dial(address string) (net.Conn, error) {
	// default dail by tcp if scheme miss
	if strings.Index(address, "://") < 0{
		address = "tcp://" + address
	}
	addrInfo, err := ParseAddress(address)
	if err != nil {
		return nil, err
	}

	if addrInfo.Host != "127.0.0.1" {
		host, err := ParseAddress(n.Address)
		if err != nil {
			return nil, err
		}
		// check if dialing address is same as its own IP
		if addrInfo.Host == host.Host {
			addrInfo.Host = "127.0.0.1"
		}
	}

	// Choose scheme.
	t, exists := n.transports.Load(addrInfo.Protocol)
	if !exists {
		err := errors.New("network: invalid protocol " + addrInfo.Protocol)
		log.Fatal().Err(err).Msg("")
	}

	var conn net.Conn
	conn, err = t.(transport.Layer).Dial(addrInfo.HostPort())
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Accept handles peer registration and processes incoming message streams.
func (n *Network) Accept(incoming net.Conn) {
	// 收到新的连接 如果无法验证这个连接的发送者则关闭这个连接
	msg, err := n.receiveMessage(incoming)
	if err != nil {
		defer incoming.Close()
		if err != errEmptyMsg {
			log.Error().Msgf("%v", err)
		}
	}
	sender := &Peer{
		PublicKey:msg.Sender.PublicKey,
		Address:msg.Sender.Address,
	}

	n.Client(sender, incoming)
}

func (n *Network) receiveLoop(peer *PeerClient, conn net.Conn) {
	for {
		msg, err := n.receiveMessage(conn)
		if err != nil {
			if err != errEmptyMsg {
				log.Error().Msgf("%v", err)
			}
			break
		}
		n.dispatchMessage(peer, msg)
	}
}

// Plugin returns a plugins proxy interface should it be registered with the
// network. The second returning parameter is false otherwise.
//
// Example: network.Plugin((*Plugin)(nil))
func (n *Network) Plugin(key interface{}) (PluginInterface, bool) {
	return n.plugins.Get(key)
}

// PrepareMessage marshals a message into a *protobuf.Message and signs it with this
// nodes private key. Errors if the message is null.
func (n *Network) PrepareMessage(ctx context.Context, message proto.Message) (*protobuf.Message, error) {
	if message == nil {
		return nil, errors.New("network: message is null")
	}

	opcode, err := opcode.GetOpcode(message)
	if err != nil {
		return nil, err
	}

	raw, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}


	msg := &protobuf.Message{
		Message: raw,
		Opcode:  uint32(opcode),
		Sender:  &protobuf.Peer{
			PublicKey:n.keys.PublicKey,
			Address:n.Address,
		},
	}

	if GetSignMessage(ctx) {
		signature, err := n.keys.Sign(
			n.opts.signaturePolicy,
			n.opts.hashPolicy,
			SerializeMessage(&Peer{
				PublicKey:msg.Sender.PublicKey,
				Address:msg.Sender.Address,
			}, raw),
		)
		if err != nil {
			return nil, err
		}
		msg.Signature = signature
	}

	return msg, nil
}

// Write asynchronously sends a message to a denoted target address.
func (n *Network) Write(id string, message *protobuf.Message) error {
	state, ok := n.ConnectionState(id)
	if !ok {
		return errors.New("network: connection does not exist")
	}

	message.MessageNonce = atomic.AddUint64(&state.messageNonce, 1)

	state.conn.SetWriteDeadline(time.Now().Add(n.opts.writeTimeout))

	err := n.sendMessage(state.writer, message, state.writerMutex)
	if err != nil {
		return err
	}
	return nil
}

// Broadcast asynchronously broadcasts a message to all peer clients.
func (n *Network) Broadcast(ctx context.Context, message proto.Message) {
	signed, err := n.PrepareMessage(ctx, message)
	if err != nil {
		log.Error().Err(err).Msg("network: failed to broadcast message")
		return
	}

	n.eachPeer(func(client *PeerClient) bool {
		err := n.Write(client.ID, signed)
		if err != nil {
			log.Warn().
				Err(err).
				Interface("peer_id", client.ID).
				Msg("failed to send message to peer")
		}
		return true
	})
}

// BroadcastByAddresses broadcasts a message to a set of peer clients denoted by their addresses.
func (n *Network) BroadcastByAddresses(ctx context.Context, message proto.Message, addresses ...string) {
	signed, err := n.PrepareMessage(ctx, message)
	if err != nil {
		return
	}

	for _, address := range addresses {
		n.Write(address, signed)
	}
}

// BroadcastByIDs broadcasts a message to a set of peer clients denoted by their peer IDs.
func (n *Network) BroadcastByIDs(ctx context.Context, message proto.Message, ids ...string) {
	signed, err := n.PrepareMessage(ctx, message)
	if err != nil {
		return
	}

	for _, id := range ids {
		n.Write(id, signed)
	}
}

// BroadcastRandomly asynchronously broadcasts a message to random selected K peers.
// Does not guarantee broadcasting to exactly K peers.
func (n *Network) BroadcastRandomly(ctx context.Context, message proto.Message, K int) {
	var addresses []string

	n.eachPeer(func(client *PeerClient) bool {
		addresses = append(addresses, client.ID)

		// Limit total amount of addresses in case we have a lot of peers.
		return len(addresses) <= K*3
	})

	// Flip a coin and shuffle :).
	rand.Shuffle(len(addresses), func(i, j int) {
		addresses[i], addresses[j] = addresses[j], addresses[i]
	})

	if len(addresses) < K {
		K = len(addresses)
	}

	n.BroadcastByAddresses(ctx, message, addresses[:K]...)
}

// Close shuts down the entire network.
func (n *Network) Close() {
	close(n.kill)

	n.eachPeer(func(client *PeerClient) bool {
		client.Close()
		return true
	})
}

func (n *Network) eachPeer(fn func(client *PeerClient) bool) {
	n.peers.Range(func(_, value interface{}) bool {
		client := value.(*PeerClient)
		return fn(client)
	})
}
