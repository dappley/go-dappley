// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package network

import (
	"bufio"
	"errors"
	"reflect"
	"sync"

	"github.com/dappley/go-dappley/network/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
)

const (
	SyncBlock    = "SyncBlock"
	SyncPeerList = "SyncPeerList"
	RequestBlock = "requestBlock"
	BroadcastTx  = "BroadcastTx"
	Unicast      = 0
	Broadcast    = 1
)

var (
	ErrInvalidMessageFormat = errors.New("invalid message format")
)

var (
	startBytes = []byte{0x7E, 0x7E}
	endBytes   = []byte{0x7F, 0x7F, 0}
)
type dapHandler func(*DapMsg, *Stream)

type Stream struct {
	peerID     peer.ID
	remoteAddr multiaddr.Multiaddr
	stream     net.Stream
	dataCh     chan []byte
	quitRdCh   chan bool
	quitWrCh   chan bool
}

func NewStream(s net.Stream) *Stream {
	return &Stream{
		s.Conn().RemotePeer(),
		s.Conn().RemoteMultiaddr(),
		s,
		make(chan []byte, 5), //TODO: Redefine the size of the channel
		make(chan bool, 1),   //two channels to stop
		make(chan bool, 1),
	}
}

func (s *Stream) Start(quitCh chan <- *Stream, dh dapHandler) {
	rw := bufio.NewReadWriter(bufio.NewReader(s.stream), bufio.NewWriter(s.stream))
	s.startLoop(rw,quitCh, dh)
}

func (s *Stream) StopStream() {
	logger.Debug("Stream Terminated! Peer Addr:", s.remoteAddr)
	s.quitRdCh <- true
	s.quitWrCh <- true
	s.stream.Close()
}

func (s *Stream) Send(data []byte) {
	s.dataCh <- data
}

func (s *Stream) startLoop(rw *bufio.ReadWriter,quitCh chan <- *Stream, dh dapHandler) {
	go s.readLoop(rw, quitCh, dh)
	go s.writeLoop(rw)
}

func readMsg(rw *bufio.ReadWriter) ([]byte, error) {
	var bytes []byte
	for {
		b, err := rw.ReadByte()

		if err != nil {
			return bytes, err
		}
		bytes = append(bytes, b)
		if containEndingBytes(bytes) {
			return bytes, nil
		}
	}
}

func (s *Stream) read(rw *bufio.ReadWriter, dh dapHandler) {
	//read stream with delimiter
	bytes, err := readMsg(rw)

	if err != nil {
		logger.Warn(err)
	}

	//TODO: How to verify the integrity of the received message
	if len(bytes) > 1 {
		dm := s.parseData(bytes)
		dh(dm, s)
	} else {
		logger.Debug("Read less than 1 byte. Stop Reading...")
		//stop the stream
		s.StopStream()
	}

}

func (s *Stream) readLoop(rw *bufio.ReadWriter, quitCh chan <- *Stream, dh dapHandler) {
	for {
		select {
		case <-s.quitRdCh:
			quitCh <- s
			logger.Debug("Stream ReadLoop Terminated!")
			return
		default:
			s.read(rw, dh)
		}
	}
}

func encodeMessage(data []byte) []byte {
	ret := append(startBytes, data...)
	ret = append(ret, endBytes...)
	return ret
}

func decodeMessage(data []byte) ([]byte, error) {
	if !(containStartingBytes(data) && containEndingBytes(data)) {
		return nil, ErrInvalidMessageFormat
	}
	return data[2 : len(data)-3], nil
}

func containStartingBytes(data []byte) bool {
	if len(data) < len(startBytes) {
		return false
	}
	return reflect.DeepEqual(data[0:len(startBytes)], startBytes)
}

func containEndingBytes(data []byte) bool {
	if len(data) < len(endBytes) {
		return false
	}
	return reflect.DeepEqual(data[(len(data)-len(endBytes)):], endBytes)
}

func (s *Stream) writeLoop(rw *bufio.ReadWriter) error {
	var mutex = &sync.Mutex{}
	for {
		select {
		case data := <-s.dataCh:
			mutex.Lock()
			//attach a delimiter byte of 0x00 to the end of the message
			rw.WriteString(string(encodeMessage(data)))
			rw.Flush()
			mutex.Unlock()
		case <-s.quitWrCh:
			logger.Debug("Stream Write Terminated!")
			return nil
		}
	}
	return nil
}

//should parse and relay
func (s *Stream) parseData(data []byte) *DapMsg {

	dataDecoded, err := decodeMessage(data)
	if err != nil {
		logger.Warn(err)
		return nil
	}

	dmpb := &networkpb.Dapmsg{}
	//unmarshal byte to proto
	if err := proto.Unmarshal(dataDecoded, dmpb); err != nil {
		logger.Info(err)
	}

	dm := &DapMsg{}
	dm.FromProto(dmpb)
	return dm

}
