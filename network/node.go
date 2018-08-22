package network

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/network/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
	"errors"
)

const (
	protocalName        = "dappley/1.0.0"
	syncPeerTimeLimitMs = 1000
)

var (
	ErrDapMsgNoCmd = errors.New("ERROR: Dappley message has no command input")
)

type Node struct {
	host      host.Host
	info      *Peer
	bc        *core.Blockchain
	blks      []*core.Block
	txnPool   core.TransactionPool
	streams   map[peer.ID]*Stream
	peerList  *PeerList
	exitCh    chan bool
}

//create new Node instance
func NewNode(bc *core.Blockchain) *Node {
	return &Node{nil,
	nil,
	bc,
	nil,
	core.TransactionPool{},
	make(map[peer.ID]*Stream, 10),
	NewPeerList(nil),
	make(chan bool, 1),
	}
}

func (n *Node) GetBlockchain() *core.Blockchain{return n.bc}
func (n *Node) GetPeerList() *PeerList{return n.peerList}

func (n *Node) Start(listenPort int) error {

	h, addr, err := createBasicHost(listenPort)
	if err != nil {
		return err
	}

	n.host = h
	n.info, err = CreatePeerFromMultiaddr(addr)

	//set streamhandler. streamHanlder function is called upon stream connection
	n.host.SetStreamHandler(protocalName, n.streamHandler)
	n.StartRequestLoop()
	return err
}

func (n *Node) StartRequestLoop() {

	go func() {
		for {
			select {
			case <-n.exitCh:
				return
			case brPars := <-n.bc.BlockPool().BlockRequestCh():
				n.RequestBlockUnicast(brPars.BlockHash, brPars.Pid)
			}
		}
	}()

}

//create basic host. Returns host object, host address and error
func createBasicHost(listenPort int) (host.Host, ma.Multiaddr, error) {

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		//libp2p.Identity(priv),
	}

	basicHost, err := libp2p.New(context.Background(), opts...)

	if err != nil {
		return nil, nil, err
	}

	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	addr := basicHost.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	logger.Info("Full Address is ", fullAddr)

	return basicHost, fullAddr, nil
}

func (n *Node) AddStreamByString(targetFullAddr string) error {
	addr, err := ma.NewMultiaddr(targetFullAddr)
	if err != nil {
		return err
	}
	return n.AddStreamMultiAddr(addr)
}

//AddStreamMultiAddr stream to the targetFullAddr address. If the targetFullAddr is nil, the node goes to listening mode
func (n *Node) AddStreamMultiAddr(targetFullAddr ma.Multiaddr) error {

	//If there is a target address, connect to that address
	if targetFullAddr != nil {

		peerInfo, err := CreatePeerFromMultiaddr(targetFullAddr)
		if err != nil {
			return err
		}

		//Add Stream
		n.AddStream(peerInfo.peerid, peerInfo.addr)
	}

	return nil
}

func (n *Node) AddStream(peerid peer.ID, targetAddr ma.Multiaddr) error {
	// We have a peer ID and a targetAddr so we add it to the peerstore
	// so LibP2P knows how to contact it
	n.host.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)

	// make a new stream
	stream, err := n.host.NewStream(context.Background(), peerid, protocalName)
	if err != nil {
		return err
	}
	// Create a buffered stream so that read and write are non blocking.
	n.streamHandler(stream)

	// Add the peer list
	if len(n.peerList.peers) >= PEERLISTMAXSIZE {
		n.peerList.RemoveOneIP(&Peer{peerid, targetAddr})
	}
	n.peerList.Add(&Peer{peerid, targetAddr})

	return nil
}

func (n *Node) streamHandler(s net.Stream) {
	// Create a buffer stream for non blocking read and write.
	logger.Info("Stream Connected! Peer Addr:", s.Conn().RemoteMultiaddr())
	// Add  the peer list
	if !n.peerList.ListIsFull() {
		n.peerList.Add(&Peer{s.Conn().RemotePeer(), s.Conn().RemoteMultiaddr()})
	}

	//start stream
	ns := NewStream(s, n)
	n.streams[s.Conn().RemotePeer()] = ns
	ns.Start()
}

func (n *Node) GetBlocks() []*core.Block { return n.blks }

func (n *Node) GetInfo() *Peer { return n.info }

func (n *Node) GetPeerMultiaddr() ma.Multiaddr {
	if n.info == nil {
		return nil
	}
	return n.info.addr
}

func (n *Node) GetPeerID() peer.ID { return n.info.peerid }

func prepareData(msgData proto.Message, cmd string) ([]byte, error){

	if cmd == "" {
		return nil, ErrDapMsgNoCmd
	}

	bytes := []byte{}
	var err error

	if msgData!=nil {
		//marshal the block to wire format
		bytes, err = proto.Marshal(msgData)
		if err != nil {
			return nil, err
		}
	}

	//build a deppley message
	dm := NewDapmsg(cmd, bytes)
	data, err := proto.Marshal(dm.ToProto())
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (n *Node) SendBlock(block *core.Block) error {
	data,err := prepareData(block.ToProto(), SyncBlock)
	if err!=nil {
		return err
	}
	n.broadcast(data)
	return nil
}

func (n *Node) SyncPeers() error {
	data,err := prepareData(n.peerList.ToProto(), SyncPeerList)
	if err!=nil {
		return err
	}
	n.broadcast(data)
	return nil
}


func (n *Node) BroadcastTxnCmd(txn *core.Transaction) error{
	data,err := prepareData(txn.ToProto(), BroadcastTxn)
	if err!=nil {
		return err
	}
	n.broadcast(data)
	return nil
}

func (n *Node) SendBlockUnicast(block *core.Block, pid peer.ID) error{
	data,err := prepareData(block.ToProto(), SyncBlock)
	if err!=nil {
		return err
	}
	n.unicast(data, pid)
	return nil
}

func (n *Node) RequestBlockUnicast(hash core.Hash, pid peer.ID) error {
	//build a deppley message
	dm := NewDapmsg(RequestBlock, hash)
	data, err := proto.Marshal(dm.ToProto())
	if err != nil {
		return err
	}
	n.unicast(data, pid)
	return nil
}

//broadcast data
func (n *Node) broadcast(data []byte) {
	for _, s := range n.streams {
		s.Send(data)
	}
}

//unicast data
func (n *Node) unicast(data []byte, pid peer.ID) {
	n.streams[pid].Send(data)
}

func (n *Node) addBlockToPool(data []byte, pid peer.ID) {

	//create a block proto
	blockpb := &corepb.Block{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(data, blockpb); err != nil {
		logger.Warn(err)
	}

	//create an empty block
	block := &core.Block{}

	//load the block with proto
	block.FromProto(blockpb)

	//add block to blockpool. Make sure this is none blocking.
	n.bc.BlockPool().Push(block, pid)
	//TODO: Delete this line. This line is solely for testing
	n.blks = append(n.blks, block)
}

func (n *Node) addTxnToPool(data []byte){

	//create a block proto
	txnpb := &corepb.Transaction{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(data, txnpb); err!=nil{
		logger.Warn(err)
	}

	//create an empty txn
	txn := &core.Transaction{}

	//load the txn with proto
	txn.FromProto(txnpb)

	//add txn to txnpool
	n.txnPool.StructPush(txn)
}


func (n *Node)addMultiPeers(data []byte){

	go func() {
		//create a peerList proto
		plpb := &networkpb.Peerlist{}

		//unmarshal byte to proto
		if err := proto.Unmarshal(data, plpb); err != nil {
			logger.Warn(err)
		}

		//create an empty peerList
		pl := &PeerList{}

		//load the block with proto
		pl.FromProto(plpb)

		//remove the node's own peer info from the list
		newpl := &PeerList{[]*Peer{n.info}}
		newpl = newpl.FindNewPeers(pl)
		//find the new added peers
		newpl = n.peerList.FindNewPeers(newpl)

		//wait for random time within the time limit
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(syncPeerTimeLimitMs)))

		//add streams for new peers
		for _, p := range newpl.GetPeerlist() {
			if !n.peerList.IsInPeerlist(p) {
				n.AddStream(p.peerid, p.addr)
			}
		}

		//add peers
		n.peerList.MergePeerlist(pl)
	}()
}

func (n *Node) sendRequestedBlock(hash []byte, pid peer.ID) {
	blockBytes, err := n.bc.DB.Get(hash)
	if err != nil {
		logger.Warn("Unable to get block data. Block request failed")
		return
	}
	block := core.Deserialize(blockBytes)
	n.SendBlockUnicast(block, pid)
}
