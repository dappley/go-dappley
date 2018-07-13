package network

import (
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-net"
	"bufio"
	"log"
	"sync"
	"github.com/multiformats/go-multiaddr"
	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/network/pb"
	"errors"
)

const(
	delimiter = 0x00
	startByte = 0x7E

	SyncBlock = "SyncBlock"
)

var(
	ErrInvalidMessageFormat = errors.New("Message format is invalid")
)

type Stream struct{
	node 		*Node
	peerID 		peer.ID
	remoteAddr	multiaddr.Multiaddr
	stream   	net.Stream
	dataCh   	chan []byte
	quitRdCh 	chan bool
	quitWrCh 	chan bool
}

func NewStream(s net.Stream, node *Node) *Stream{
	return &Stream{	node,
					s.Conn().RemotePeer(),
					s.Conn().RemoteMultiaddr(),
					s,
					make(chan []byte, 5), 	//TODO: Redefine the size of the channel
					make(chan bool, 1), 		//two channels to stop
					make(chan bool, 1),
	}
}

func (s *Stream) Start(){
	rw := bufio.NewReadWriter(bufio.NewReader(s.stream), bufio.NewWriter(s.stream))
	s.startLoop(rw)
}

func (s *Stream) startLoop(rw *bufio.ReadWriter){
	go s.readLoop(rw)
	go s.writeLoop(rw)
}


func (s *Stream) read(rw *bufio.ReadWriter){
	//read stream with delimiter
	bytes,err := rw.ReadBytes(delimiter)

	if err != nil {
		log.Println(err)
	}

	//TODO: How to verify the integrity of the received message
	//if the string is not empty
	if len(bytes) > 1 {
		//prase data
		log.Println("Received Data:", bytes)
		s.parseData(bytes)
	}else{

		//stop the stream
		s.StopStream()
	}
}

func (s *Stream) readLoop(rw *bufio.ReadWriter) {
	for {
		select{
		case <- s.quitRdCh:
			log.Println("Stream ReadLoop Terminated!")
			return
		default:
			s.read(rw)
		}
	}
}

func encodeMessage(data []byte) []byte{
	startArr := []byte{startByte, startByte}
	data = append(startArr,data...)
	return append(data, delimiter)
}

func decodeMessage(data []byte) ([]byte,error){
	if data[0]!=startByte || data[1]!=startByte || data[len(data)-1]!= delimiter {
		return nil,ErrInvalidMessageFormat
	}
	return data[2:len(data)-1],nil
}

func (s *Stream) writeLoop(rw *bufio.ReadWriter) error{
	var mutex = &sync.Mutex{}
	for{
		select{
		case data := <- s.dataCh:
			mutex.Lock()
			//attach a delimiter byte of 0x00 to the end of the message
			rw.WriteString(string(encodeMessage(data)))
			rw.Flush()
			mutex.Unlock()
		case <- s.quitWrCh:
			log.Println("Stream Write Terminated!")
			return nil
		}
	}
	return nil
}

func (s *Stream) StopStream(){
	log.Println("Stream Terminated! Peer Addr:", s.remoteAddr)
	s.quitRdCh <- true;
	s.quitWrCh <- true;
	s.stream.Close()
	delete(s.node.streams, s.peerID)
}

func (s *Stream) Send(data []byte){
	s.dataCh <- data
}

func (s *Stream) parseData(data []byte){

	data,err := decodeMessage(data)
	if err!=nil {
		log.Println(err)
		return
	}

	dmpb := &networkpb.Dapmsg{}
	//unmarshal byte to proto
	if err := proto.Unmarshal(data, dmpb); err!=nil{
		log.Println(err)
	}

	dm := &Dapmsg{}
	dm.FromProto(dmpb)
	switch(dm.GetCmd()){
	case SyncBlock:
		log.Print("Received SyncBlock command from:", s.remoteAddr)
		s.addBlockToPool(dm.GetData())
	default:
		log.Print("Received invalid command from:", s.remoteAddr)
	}

}

func (s *Stream) addBlockToPool(data []byte){

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

	//add block to blockpool
	s.node.bc.BlockPool().Push(block)
	//TODO: Delete this line. This line is solely for testing
	s.node.blks = append(s.node.blks, block)
}