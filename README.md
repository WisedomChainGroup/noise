# Noise

[![GoDoc][1]][2] [![Discord][7]][8] [![MIT licensed][5]][6] [![Build Status][9]][10] [![Go Report Card][11]][12] [![Coverage Statusd][13]][14]

[1]: https://godoc.org/github.com/perlin-network/noise?status.svg
[2]: https://godoc.org/github.com/perlin-network/noise
[5]: https://img.shields.io/badge/license-MIT-blue.svg
[6]: LICENSE
[7]: https://img.shields.io/discord/458332417909063682.svg
[8]: https://discord.gg/dMYfDPM
[9]: https://travis-ci.org/perlin-network/noise.svg?branch=master
[10]: https://travis-ci.org/perlin-network/noise
[11]: https://goreportcard.com/badge/github.com/perlin-network/noise
[12]: https://goreportcard.com/report/github.com/perlin-network/noise
[13]: https://codecov.io/gh/perlin-network/noise/branch/master/graph/badge.svg
[14]: https://codecov.io/gh/perlin-network/noise


<img align="right" width=400 src="media/chat.gif">

**noise** is an opinionated, easy-to-use P2P network stack for
*decentralized applications, and cryptographic protocols* written in
[Go](https://golang.org/) by Perlin Network.

**noise** is made to be robust, developer-friendly, performant, secure, and
cross-platform across multitudes of devices by making use of well-tested,
production-grade dependencies.

## Features

- Modular design with interfaces for establishing connections, verifying identity/authorization, and sending/receiving messages.
- Real-time, bidirectional streaming between peers via TCP and
  [Protobufs](https://developers.google.com/protocol-buffers/).
- NAT traversal/automated port forwarding (NAT-PMP, UPnP).
- [NaCL/Ed25519](https://tweetnacl.cr.yp.to/) scheme for peer identities and
  signatures.
- [S/Kademlia](https://ieeexplore.ieee.org/document/4447808) peer identity and discovery.
- Request/Response and Messaging RPC.
- Logging via [zerolog](https://github.com/rs/zerolog/log).
- Plugin system via services.

## Setup

### Dependencies

 - [Protobuf compiler](https://github.com/google/protobuf/releases) (protoc)
 - [Go 1.11](https://golang.org/dl/) (go)

```bash
# enable go modules: https://github.com/golang/go/wiki/Modules
export GO111MODULE=on

# download the dependencies to vendor folder for tools
go mod vendor

# generate necessary code files
go get -u github.com/gogo/protobuf/protoc-gen-gogofaster
# tested with version v1.1.1 (636bf0302bc95575d69441b25a2603156ffdddf1)
go get -u github.com/golang/mock/mockgen
# tested with v1.1.1 (c34cdb4725f4c3844d095133c6e40e448b86589b)
go generate ./...

# run an example
[terminal 1] go run examples/chat/main.go -port 3000 -private_key aefd86bdfefe4e2eca563782682d7612a856191b48844687fec1c8a22dc70f220da80160d6b3686d66a4ad8ac692a322043b0239302c5037988d4bb1e41830f1
[terminal 2] go run examples/chat/main.go -port 3001 -peers 0da80160d6b3686d66a4ad8ac692a322043b0239302c5037988d4bb1e41830f1=localhost:3000
[terminal 3] go run examples/chat/main.go -port 3002 -peers 0da80160d6b3686d66a4ad8ac692a322043b0239302c5037988d4bb1e41830f1=localhost:3000

# run test cases
go test -v -count=1 -race ./...

# run test cases short
go test -v -count=1 -race -short ./...
```

## Usage

Noise is designed to be modular and splits the networking, node identity, and message sending/receiving into separate interfaces.

Noise provides an identity adapter which randomly generates cryptographic public/private keys using the Ed25519 signature scheme.

```go
// Create a new identity adapter and get the Ed25519 keypair.
idAdapter := base.NewIdentityAdapter()
keys := idAdapter.GetKeyPair()

// Create an identity adapter from an existing hex-encoded private key.
keys, _ := crypto.FromPrivateKey(ed25519.New(), "4d5333a68e3a96d0ad935cb6546b97bbb0c0771acf76c868a897f65dad0b7933e1442970cce57b7a35e1803e0e8acceb04dc6abf8a73df52e808ab5d966113ac")
idAdapter = base.NewIdentityAdapterFromKeypair(keys)

// Print out loaded public/private keys.
log.Info().
    Str("private_key", idAdapter.GetKeyPair().PrivateKeyHex()).
    Msg("")
log.Info().
    Str("public_key", idAdapter.GetKeyPair().PublicKeyHex()).
    Msg("")
log.Info().
    Str("node_id", idAdapter.MyIdentity()).
    Msg("")
```

Now that you have your keys, we can start listening and handling messages from
incoming peers.

```go
// setup the node with a new identity adapter
idAdapter := base.NewIdentityAdapter()
node := protocol.NewNode(
    protocol.NewController(),
    idAdapter,
)

// Create the listener which peers will use to connect to
// you. For example, set the host part to `localhost` if you are testing
// locally, or your public IP address if you are connected to the internet
// directly.
listener, _ := net.Listen("tcp", "localhost:3000")

// Create the dialer for the connection adapter
func dialer(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

// ... add services for the node here.

// Start the node.
node.Start()
```

See `examples/getting_started` for a full working example to get started with.

## Services

Services are a way to interface with the lifecycle of your network.


```go
type YourAwesomeService struct {
	protocol.Service
	Mailbox chan *messages.BasicMessage
}

func (state *YourAwesomeService) Startup(net *network.Network)              {}
func (state *YourAwesomeService) Receive(ctx context.Context, request *Message) (*MessageBody, error)  { return nil }
func (state *YourAwesomeService) Cleanup(node *Node)              {}
func (state *YourAwesomeService) PeerConnect(id []byte)    {}
func (state *YourAwesomeService) PeerDisconnect(id []byte) {}
```

They are registered through `protocol.Node` through the following:

```go
node := protocol.NewNode(
    protocol.NewController(),
    base.NewIdentityAdapter(),
)

// Add service.
service := &BasicService{
    Mailbox: make(chan *messages.BasicMessage, 1),
}
node.AddService(service)
```

**noise** comes with three plugins: `discovery.Plugin`, `backoff.Plugin` and
`nat.Plugin`.

```go
// Enables peer discovery through the network.
// Check documentation for more info.
builder.AddPlugin(new(discovery.Plugin))

// Enables exponential backoff upon peer disconnection.
// Check documentation for more info.
builder.AddPlugin(new(backoff.Plugin))

// Enables automated NAT traversal/port forwarding for your node.
// Check documentation for more info.
nat.RegisterPlugin(builder)
```

Make sure to register `discovery.Plugin` if you want to make use of automatic
peer discovery within your application.

## Handling Messages

All messages that pass through **noise** are serialized/deserialized as
[protobufs](https://developers.google.com/protocol-buffers/). If you want to
use a new message type for your application, you must first register your
message type.

```go
opcode.RegisterMessageType(opcode.Opcode(1000), &MyNewProtobufMessage{})
```

On a spawned `us-east1-b` Google Cloud (GCP) cluster comprised of 8
`n1-standard-1` (1 vCPU, 3.75GB memory) instances, **noise** is able to sign,
send, receive, verify, and process a total of ~10,000 messages per second.

Once you have modeled your messages as protobufs, you may process and receive
them over the network by creating a plugin and overriding the
`Receive(ctx *PluginContext)` method to process specific incoming message types.

Here's a simple example:

```go
// An example chat plugin that will print out a formatted chat message.
type ChatPlugin struct{ *network.Plugin }

func (state *ChatPlugin) Receive(ctx *network.PluginContext) error {
    switch msg := ctx.Message().(type) {
        case *messages.ChatMessage:
            log.Info().Msgf("<%s> %s", ctx.Client().ID.Address, msg.Message)
    }
    return nil
}

// Register plugin to *network.Builder.
builder.AddPlugin(new(ChatPlugin))
```

Through a `ctx *network.PluginContext`, you can access flexible methods to
customize how you handle/interact with your peer network. All messages are
signed and verified with one's cryptographic keys.

```go
// Reply with a message should the incoming message be a request.
err := ctx.Reply(message here)
if err != nil {
    return err
}

// Get an instance of your own nodes ID.
self := ctx.Self()

// Get an instance of the peers ID which sent the message.
sender := ctx.Sender()

// Get access to an instance of the peers client.
client := ctx.Client()

// Get access to your own nodes network instace.
net := ctx.Network()
```

Check out our documentation and look into the `examples/` directory to find out
more.

## Contributions

We at Perlin love reaching out to the open-source community and are open to
accepting issues and pull-requests.

For all code contributions, please ensure they adhere as close as possible to
the following guidelines:

1. **Strictly** follows the formatting and styling rules denoted
   [here](https://github.com/golang/go/wiki/CodeReviewComments).
2. Commit messages are in the format `module_name: Change typed down as a sentence.`
   This allows our maintainers and everyone else to know what specific code
   changes you wish to address.
    - `network: Added in message broadcasting methods.`
    - `builders/network: Added in new option to address PoW in generating peer IDs.`
3. Consider backwards compatibility. New methods are perfectly fine, though
   changing the `network.Builder` pattern radically for example should only be
   done should there be a good reason.

If you...

1. love the work we are doing,
2. want to work full-time with us,
3. or are interested in getting paid for working on open-source projects

... **we're hiring**.

To grab our attention, just make a PR and start contributing.
