package network

import (
	"context"
	"fmt"
	"log"
	"bufio"
	"sync"

	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/core"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/gogo/protobuf/proto"
)

const(
	protocalName = "dappley/1.0.0"
 	delimiter = 0x00
)

type Node struct{
	host     host.Host
	addr     ma.Multiaddr
	bc       *core.Blockchain
	blks     []*core.Block
	dataCh   chan []byte
	quitRdCh chan bool
	quitWrCh chan bool
}

var writeLoopCount = int(0)
var readLoopCount = int(0)


//create new Node instance
func NewNode(bc *core.Blockchain) *Node{

	return &Node{nil,
	nil,
	bc,
	nil,
	make(chan []byte, 5), 	//TODO: Redefine the size of the channel
	make(chan bool, 1), 		//two channels to stop
	make(chan bool, 1),
	}
}

func (n *Node) Start(listenPort int) error{
	h,addr,err :=createBasicHost(listenPort)
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
		//libp2p.ListenAddrs(multiaddrs[0]),
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
		fmt.Println(n.host.Peerstore().Peers())



		// make a new stream
		stream, err := n.host.NewStream(context.Background(), peerid, protocalName)
		log.Println("Opening stream to peer:", stream.Conn().RemoteMultiaddr())
		if err != nil {
			return err
		}

		// Create a buffered stream so that read and write are non blocking.
		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
		n.startLoop(rw)
	}

	return nil
}

func (n *Node) startLoop(rw *bufio.ReadWriter){
	go n.readLoop(rw)
	go n.writeLoop(rw)
}

func (n *Node) read(rw *bufio.ReadWriter){
	//read stream with delimiter
	bytes,err := rw.ReadBytes(delimiter)

	if err != nil {
		log.Println(err)
	}

	//TODO: How to verify the integrity of the received message
	//if the string is not empty
	if len(bytes) > 1 {

		//get rid of the delimiter
		bytes = bytes[:len(bytes)-1]
		log.Println("Received Data:", bytes)

		//create a block proto
		blockpb := &corepb.Block{}

		//unmarshal byte to proto
		if err := proto.Unmarshal(bytes, blockpb); err!=nil{
			log.Println(err)
		}

		//create an empty block
		block := &core.Block{}

		//load the block with proto
		block.FromProto(blockpb)

		//TODO: add block to blockchain
		n.bc.BlockPool().Push(block)
		//add the block to the buffer pool (for testing purpose)
		n.blks = append(n.blks, block)
	}else{
		//stop the stream
		log.Println("Empty Byte Detected.Probably due to stream reset")
		n.StopStream()
	}
}

func (n *Node) readLoop(rw *bufio.ReadWriter) {
	readLoopCount++
	log.Println("ReadLoopLoopCount:",readLoopCount)
	for {
		select{
			case <- n.quitRdCh:
				log.Println("Stream ReadLoop Terminated!")
				readLoopCount--
				log.Println("ReadLoopLoopCount:",readLoopCount)
				return
			default:
				n.read(rw)
		}
	}
}

func (n *Node) StopStream(){
	log.Println("Stream Terminated")
	n.quitRdCh <- true;
	n.quitWrCh <- true;
}

func (n *Node) writeLoop(rw *bufio.ReadWriter) error{
	var mutex = &sync.Mutex{}
	writeLoopCount++
	log.Println("WriteLoopCount:",writeLoopCount)
	for{
		select{
			case data := <- n.dataCh:
				mutex.Lock()
				//attach a delimiter byte of 0x00 to the end of the message
				rw.WriteString(string(append(data, delimiter)))
				rw.Flush()
				mutex.Unlock()
			case <- n.quitWrCh:
				log.Println("Stream Write Terminated!")
				writeLoopCount--
				log.Println("WriteLoopCount:",writeLoopCount)
				log.Println("ReadLoopLoopCount:",readLoopCount)
				return nil
		}
	}
	return nil
}

func (n *Node) streamHandler(s net.Stream){
	// Create a buffer stream for non blocking read and write.
	log.Println("Got a new STREAM from peer:", s.Conn().RemoteMultiaddr())
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	n.startLoop(rw)
}

func (n *Node) GetBlocks() []*core.Block {return n.blks}

func (n *Node) GetMultiaddr() ma.Multiaddr { return n.addr}

func (n *Node) SendBlock(block *core.Block) error{
	//marshal the block to wire format
	bytes, err :=proto.Marshal(block.ToProto())
	if err != nil {
		return err
	}

	log.Println("Sending data:",bytes)
	n.dataCh <- bytes
	return nil
}