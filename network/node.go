package network

import (
	"context"
	"fmt"
	"log"

	"github.com/dappley/go-dappley/core"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/core/pb"
)

const(
	protocalName = "dappley/1.0.0"
)

type Node struct{
	host     	host.Host
	addr     	ma.Multiaddr
	bc       	*core.Blockchain
	blks 	 	[]*core.Block
	blockpool 	[]*core.Block
	streams  	map[peer.ID]*Stream
	peerlist	*Peerlist
}

var writeLoopCount = int(0)
var readLoopCount = int(0)


//create new Node instance
func NewNode(bc *core.Blockchain) *Node{
	return &Node{nil,
	nil,
	bc,
	nil,
	nil,
	make(map[peer.ID]*Stream, 10),
	NewPeerlist(nil),
	}
}

func (n *Node) Start(listenPort int) error{
	h,addr,err := createBasicHost(listenPort)
	if err != nil {
		return err
	}

	n.host = h
	n.addr = addr

	//set streamhandler. streamHanlder function is called upon stream connection
	n.host.SetStreamHandler(protocalName, n.streamHandler)
	return nil
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
	log.Printf("Full Address is %s\n", fullAddr)

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

		//get pid
		pid, err := targetFullAddr.ValueForProtocol(ma.P_IPFS)
		if err != nil {
			return err
		}

		//get peer id
		peerid, err := peer.IDB58Decode(pid)
		if err != nil {
			return err
		}

		// Decapsulate the /ipfs/<peerID> part from the targetFullAddr
		// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
		targetPeerAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
		targetAddr := targetFullAddr.Decapsulate(targetPeerAddr)

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

		// Add the full addr to the peer list
		n.peerlist.Add(targetFullAddr)

	}

	return nil
}

func (n *Node) streamHandler(s net.Stream){
	// Create a buffer stream for non blocking read and write.
	log.Println("Stream Connected! Peer Addr:", s.Conn().RemoteMultiaddr())
	ns := NewStream(s, n)
	n.streams[s.Conn().RemotePeer()] = ns
	ns.Start()
}

func (n *Node) GetBlocks() []*core.Block { return n.blks }

func (n *Node) GetMultiaddr() ma.Multiaddr { return n.addr}

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
		log.Println(err)
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

