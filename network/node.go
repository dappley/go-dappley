package network

import (
	"context"
	"fmt"
	"github.com/dappley/go-dappley/core"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/network/pb"
	"time"
	"math/rand"
)

const(
	protocalName 			= "dappley/1.0.0"
	syncPeerTimeLimitMs 	= 1000
)

type Node struct{
	host      host.Host
	info      *Peer
	bc        *core.Blockchain
	blks      []*core.Block
	blockpool []*core.Block
	streams   map[peer.ID]*Stream
	peerlist  *PeerList
}

//create new Node instance
func NewNode(bc *core.Blockchain) *Node{
	return &Node{nil,
	nil,
	bc,
	nil,
	nil,
	make(map[peer.ID]*Stream, 10),
	NewPeerList(nil),
	}
}

func (n *Node) Start(listenPort int) error{
	h,addr,err := createBasicHost(listenPort)
	if err != nil {
		return err
	}

	n.host = h
	n.info, err = CreatePeerFromMultiaddr(addr)

	//set streamhandler. streamHanlder function is called upon stream connection
	n.host.SetStreamHandler(protocalName, n.streamHandler)
	return err
}

//create basic host. Returns host object, host address and error
func createBasicHost(listenPort int) (host.Host, ma.Multiaddr, error){

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
	logger.Info("Full Address is %s\n", fullAddr)

	return basicHost,fullAddr, nil
}

func (n *Node) AddStreamString(targetFullAddr string) error{
	addr, err:=ma.NewMultiaddr(targetFullAddr)
	if err!= nil {
		return err
	}
	return n.AddStreamMultiAddr(addr)
}


//AddStreamMultiAddr stream to the targetFullAddr address. If the targetFullAddr is nil, the node goes to listening mode
func (n *Node) AddStreamMultiAddr(targetFullAddr ma.Multiaddr) error{

	//If there is a target address, connect to that address
	if targetFullAddr != nil {

		peerInfo, err := CreatePeerFromMultiaddr(targetFullAddr)
		if err != nil {
			return err
		}

		//Add Stream
		n.AddStream(peerInfo.peerid,peerInfo.addr)
	}

	return nil
}

func (n *Node) AddStream(peerid peer.ID, targetAddr ma.Multiaddr) error{
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
	n.peerlist.Add(&Peer{peerid,targetAddr})

	return nil
}

func (n *Node) streamHandler(s net.Stream){
	// Create a buffer stream for non blocking read and write.
	logger.Info("Stream Connected! Peer Addr:", s.Conn().RemoteMultiaddr())
	// Add  the peer list
	n.peerlist.Add(&Peer{s.Conn().RemotePeer(),s.Conn().RemoteMultiaddr()})
	//start stream
	ns := NewStream(s, n)
	n.streams[s.Conn().RemotePeer()] = ns
	ns.Start()
}

func (n *Node) GetBlocks() []*core.Block { return n.blks }

func (n *Node) GetInfo() *Peer { return n.info }

func (n *Node) GetPeerMultiaddr() ma.Multiaddr {return n.info.addr}

func (n *Node) GetPeerID() peer.ID {return n.info.peerid}

func (n *Node) SendBlock(block *core.Block) error{
	//marshal the block to wire format
	bytes, err :=proto.Marshal(block.ToProto())
	if err != nil {
		return err
	}

	//build a deppley message
	dm := NewDapmsg(SyncBlock,bytes)
	data, err :=proto.Marshal(dm.ToProto())
	if err != nil {
		return err
	}
	//log.Println("Sending Data Request Received:",bytes)
	n.broadcast(data)
	return nil
}

func (n *Node) SyncPeers() error{
	//marshal the peerlist to wire format
	bytes, err :=proto.Marshal(n.peerlist.ToProto())
	if err != nil {
		return err
	}

	//build a deppley message
	dm := NewDapmsg(SyncPeerList,bytes)
	data, err :=proto.Marshal(dm.ToProto())
	if err != nil {
		return err
	}
	//log.Println("Sending Data Request Received:",bytes)
	n.broadcast(data)
	return nil
}

//broadcast data
func (n *Node) broadcast(data []byte){
	//log.Println("Broadcasting to",len(n.streams), "peer(s)...")
	for _,s := range n.streams{
		s.Send(data)
	}
}

func (n *Node) addBlockToPool(data []byte){

	//create a block proto
	blockpb := &corepb.Block{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(data, blockpb); err!=nil{
		logger.Warn(err)
	}

	//create an empty block
	block := &core.Block{}

	//load the block with proto
	block.FromProto(blockpb)

	//add block to blockpool. Make sure this is none blocking.
	n.bc.BlockPool().Push(block)
	//TODO: Delete this line. This line is solely for testing
	n.blks = append(n.blks, block)
}

func (n *Node)addMultiPeers(data []byte){

	go func() {
		//create a peerlist proto
		plpb := &networkpb.Peerlist{}

		//unmarshal byte to proto
		if err := proto.Unmarshal(data, plpb); err != nil {
			logger.Warn(err)
		}

		//create an empty peerlist
		pl := &PeerList{}

		//load the block with proto
		pl.FromProto(plpb)

		//remove the node's own peer info from the list
		newpl := &PeerList{[]*Peer{n.info}}
		newpl = newpl.FindNewPeers(pl)
		//find the new added peers
		newpl = n.peerlist.FindNewPeers(newpl)

		//wait for random time within the time limit
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(syncPeerTimeLimitMs)) )

		//add streams for new peers
		for _, p := range newpl.GetPeerlist() {
			if !n.peerlist.IsInPeerlist(p){
				n.AddStream(p.peerid, p.addr)
			}
		}

		//add peers
		n.peerlist.MergePeerlist(pl)
	}()
}
