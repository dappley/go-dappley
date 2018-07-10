package network

import (
	"context"
	"fmt"
	"log"
	"bufio"
	"sync"

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

const protocalName = "dappley/1.0.0"

type Network struct{
	rw *bufio.ReadWriter
	blks []*core.Block
}

func NewNetwork() *Network{

	return &Network{}
}

func CreateBasicHost(listenPort int) (host.Host, ma.Multiaddr, error){
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", listenPort)),
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

func (net *Network) Setup(host host.Host, targetIPFSaddr ma.Multiaddr) error{

	host.SetStreamHandler(protocalName, net.streamHandler)

	//if the target addr is nil, go to listening mode.
	//If there is a target address, connect to that address
	if targetIPFSaddr != nil {

		//get pid
		pid, err := targetIPFSaddr.ValueForProtocol(ma.P_IPFS)
		if err != nil {
			return err
		}

		//get peer id
		peerid, err := peer.IDB58Decode(pid)
		if err != nil {
			return err
		}

		// Decapsulate the /ipfs/<peerID> part from the targetIPFSaddr
		// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
		targetPeerAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
		targetAddr := targetIPFSaddr.Decapsulate(targetPeerAddr)

		// We have a peer ID and a targetAddr so we add it to the peerstore
		// so LibP2P knows how to contact it
		host.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)

		log.Println("opening stream")
		// make a new stream
		s, err := host.NewStream(context.Background(), peerid, protocalName)
		if err != nil {
			return err
		}

		// Create a buffered stream so that read is non blocking.
		net.rw = bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		go net.Read()
	}

	return nil
}

func (net *Network) Read() {
	for {
		//read stream with delimiter of byte 0x00.
		str, err := net.rw.ReadString(byte(0))
		//get rid of the delimiter byte (last byte)
		str = str[:len(str)-1]

		if err != nil {
			log.Println(err)
		}

		//TODO: How to verify the integrity of the received message
		//if the string is not empty
		if str != "" {
			//create a block proto
			blockpb := &corepb.Block{}
			//unmarshal byte to proto
			if err := proto.Unmarshal([]byte(str), blockpb); err!=nil{
				log.Println(err)
			}
			//create an empty block
			block := &core.Block{}
			//load the block with proto
			block.FromProto(blockpb)

			//TODO: add blockpb to blockchain
			//add the block to the buffer pool
			net.blks = append(net.blks, block)
		}

	}
}

func (net *Network) Write(block *core.Block) error{
	var mutex = &sync.Mutex{}

	//marshal the block to wire format
	bytes, err :=proto.Marshal(block.ToProto())

	if err != nil {
		return err
	}

	mutex.Lock()
	//attach a delimiter byte of 0x00 to the end of the message
	net.rw.WriteString(fmt.Sprintf("%s\n", string(append(bytes, 0x00))))
	net.rw.Flush()
	mutex.Unlock()
	return nil
}

func (net *Network) streamHandler(s net.Stream){
	// Create a buffer stream for non blocking read and write.
	net.rw = bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	go net.Read()
}

func (net *Network) GetBlocks() []*core.Block{
	return net.blks
}