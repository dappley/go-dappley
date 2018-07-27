package network

import (
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-net"
	"bufio"
	"sync"
	"github.com/multiformats/go-multiaddr"
	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/network/pb"
	"errors"
	logger "github.com/sirupsen/logrus"
	"reflect"
)

const(
	SyncBlock 		= "SyncBlock"
	SyncPeerList 	= "SyncPeerList"
	RequestBlock	= "requestBlock"
)

var(
	ErrInvalidMessageFormat = errors.New("Message format is invalid")
)

var(
	startBytes = []byte{0x7E, 0x7E}
	endBytes = []byte{0x7F, 0x7F, 0}
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

func readMsg(rw *bufio.ReadWriter) ([]byte,error){
	var bytes []byte
	for{
		b, err := rw.ReadByte()

		if err!=nil {
			return bytes,err
		}
		bytes = append(bytes, b)
		if containEndingBytes(bytes) {
			return bytes,nil
		}
	}
}

func (s *Stream) read(rw *bufio.ReadWriter){
	//read stream with delimiter
	bytes,err := readMsg(rw)

	if err != nil {
		logger.Warn(err)
	}

	//TODO: How to verify the integrity of the received message
	//if the string is not empty
	if len(bytes) > 1 {
		//prase data
		logger.Debug("Received Data:", bytes)
		s.parseData(bytes)
	}else{
		logger.Debug("Read less than 1 byte. Stop Reading...")
		//stop the stream
		s.StopStream()
	}

}

func (s *Stream) readLoop(rw *bufio.ReadWriter) {
	for {
		select{
		case <- s.quitRdCh:
			logger.Debug("Stream ReadLoop Terminated!")
			return
		default:
			s.read(rw)
		}
	}
}

func encodeMessage(data []byte) []byte{
	ret := append(startBytes,data...)
	ret = append(ret, endBytes...)
	return ret
}

func decodeMessage(data []byte) ([]byte,error){
	if !(containStartingBytes(data) && containEndingBytes(data)) {
		return nil,ErrInvalidMessageFormat
	}
	return data[2:len(data)-3],nil
}

func containStartingBytes(data []byte) bool{
	if len(data) < len(startBytes){
		return false
	}
	return reflect.DeepEqual(data[0:len(startBytes)], startBytes)
}

func containEndingBytes(data []byte) bool{
	if len(data) < len(endBytes){
		return false
	}
	return 	reflect.DeepEqual(data[(len(data)-len(endBytes)):], endBytes)
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
			logger.Debug("Stream Write Terminated!")
			return nil
		}
	}
	return nil
}

func (s *Stream) StopStream(){
	logger.Debug("Stream Terminated! Peer Addr:", s.remoteAddr)
	s.quitRdCh <- true
	s.quitWrCh <- true
	s.stream.Close()
	delete(s.node.streams, s.peerID)
}

func (s *Stream) Send(data []byte){
	s.dataCh <- data
}

func (s *Stream) parseData(data []byte){

	data,err := decodeMessage(data)
	if err!=nil {
		logger.Warn(err)
		return
	}

	dmpb := &networkpb.Dapmsg{}
	//unmarshal byte to proto
	if err := proto.Unmarshal(data, dmpb); err!=nil{
		logger.Info(err)
	}

	dm := &Dapmsg{}
	dm.FromProto(dmpb)
	switch(dm.GetCmd()) {
	case SyncBlock:
		logger.Debug("Received ",SyncBlock," command from:", s.remoteAddr)
		s.node.addBlockToPool(dm.GetData(),s.peerID)
	case SyncPeerList:
		logger.Debug("Received ",SyncPeerList," command from:", s.remoteAddr)
		s.node.addMultiPeers(dm.GetData())
	case RequestBlock:
		logger.Debug("Received ",RequestBlock," command from:", s.remoteAddr)
		s.node.sendRequestedBlock(dm.GetData(),s.peerID)
	default:
		logger.Debug("Received invalid command from:", s.remoteAddr)
	}

}

