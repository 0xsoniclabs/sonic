package gossip

import (
	"math/big"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	notify "github.com/ethereum/go-ethereum/event"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/emitter"
	"github.com/0xsoniclabs/sonic/inter"
)

// Constants to match up protocol versions and messages
const (
	_FTM62    = 62
	_Sonic_64 = 64
	_Sonic_65 = 65
)

// ProtocolName is the official short name of the protocol used during capability negotiation.
const ProtocolName = "opera"

// ProtocolVersions are the supported versions of the protocol (first is primary).
var ProtocolVersions = []uint{
	_Sonic_65,
	_Sonic_64,
	_FTM62,
}

// protocolLengths are the number of implemented message corresponding to different protocol versions.
var protocolLengths = map[uint]uint64{
	_Sonic_65: EndPointUpdateMsg + 1,
	_Sonic_64: PeerInfosMsg + 1,
	_FTM62:    EventsStreamResponse + 1,
}

const protocolMaxMsgSize = inter.ProtocolMaxMsgSize // Maximum cap on the size of a protocol message

// protocol message codes
const (
	HandshakeMsg = 0

	// Signals about the current synchronization status.
	// The current peer's status is used during packs downloading,
	// and to estimate may peer be interested in the new event or not
	// (based on peer's epoch).
	ProgressMsg = 1

	EvmTxsMsg         = 2
	NewEvmTxHashesMsg = 3
	GetEvmTxsMsg      = 4

	// Non-aggressive events propagation. Signals about newly-connected
	// batch of events, sending only their IDs.
	NewEventIDsMsg = 5

	// Request the batch of events by IDs
	GetEventsMsg = 6
	// Contains the batch of events.
	// May be an answer to GetEventsMsg, or be sent during aggressive events propagation.
	EventsMsg = 7

	// Request a range of events by a selector
	RequestEventsStream = 8
	// Contains the requested events by RequestEventsStream
	EventsStreamResponse = 9

	// Request the list of known peers and their information.
	GetPeerInfosMsg = 10
	// Contains the list of known peers and their information.
	PeerInfosMsg = 11

	// Request the enode of the peer identifying its public end-point.
	GetEndPointMsg = 12
	// Contains the enode including the public end-point of the sender.
	EndPointUpdateMsg = 13
)

type errCode int

const (
	ErrMsgTooLarge = iota
	ErrDecode
	ErrInvalidMsgCode
	ErrProtocolVersionMismatch
	ErrNetworkIDMismatch
	ErrGenesisMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
	ErrSuspendedPeer
	ErrEmptyMessage = 0xf00
)

func (e errCode) String() string {
	return errorToString[int(e)]
}

// XXX change once legacy code is out
var errorToString = map[int]string{
	ErrMsgTooLarge:             "Message too long",
	ErrDecode:                  "Invalid message",
	ErrInvalidMsgCode:          "Invalid message code",
	ErrProtocolVersionMismatch: "Protocol version mismatch",
	ErrNetworkIDMismatch:       "NetworkId mismatch",
	ErrGenesisMismatch:         "Genesis object mismatch",
	ErrNoStatusMsg:             "No status message",
	ErrExtraStatusMsg:          "Extra status message",
	ErrSuspendedPeer:           "Suspended peer",
	ErrEmptyMessage:            "Empty message",
}

type TxPool interface {
	emitter.TxPool
	SubscribeNewTxsNotify(chan<- evmcore.NewTxsNotify) notify.Subscription
	// AddRemotes should add the given transactions to the pool.
	AddRemotes([]*types.Transaction) []error
	AddLocals(txs []*types.Transaction) []error
	AddLocal(tx *types.Transaction) error

	Get(common.Hash) *types.Transaction

	OnlyNotExisting(hashes []common.Hash) []common.Hash
	SampleHashes(max int) []common.Hash

	Nonce(addr common.Address) uint64
	Stats() (int, int)
	Content() (map[common.Address]types.Transactions, map[common.Address]types.Transactions)
	ContentFrom(addr common.Address) (types.Transactions, types.Transactions)
	GasPrice() *big.Int
}

// handshakeData is the network packet for the initial handshake message
type handshakeData struct {
	ProtocolVersion uint32
	NetworkID       uint64
	Genesis         common.Hash
}

// PeerProgress is synchronization status of a peer
type PeerProgress struct {
	Epoch           consensus.Epoch
	LastBlockIdx    consensus.BlockID
	LastBlockLeader consensus.EventHash
	// Currently unused
	HighestLamport consensus.Lamport
}

type dagChunk struct {
	SessionID uint32
	Done      bool
	IDs       consensus.EventHashes
	Events    inter.EventPayloads
}

type peerInfo struct {
	Enode string
}

type peerInfoMsg struct {
	Peers []peerInfo
}
