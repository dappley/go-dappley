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
	"github.com/dappley/go-dappley/network/pb"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
	"math/big"
	"reflect"
	"time"
)

const (
	lengthByteLength       = 8
	startByteLength        = 2
	checkSumLength         = 1
	headerLength           = lengthByteLength + startByteLength + checkSumLength
	highPriorityChLength   = 1024 * 4
	normalPriorityChLength = 1024 * 4
	WriteChTotalLength     = highPriorityChLength + normalPriorityChLength
)

const (
	HighPriorityCommand = iota
	NormalPriorityCommand
)

var (
	ErrInvalidMessageFormat = errors.New("invalid message format")
	ErrLengthTooShort       = errors.New("message length is too short")
	ErrFragmentedData       = errors.New("fragmented data")
	ErrCheckSumIncorrect    = errors.New("incorrect checksum")
)

var (
	startBytes = []byte{0x7E, 0x7E}
)

type Stream struct {
	peerID                peer.ID
	remoteAddr            multiaddr.Multiaddr
	stream                network.Stream
	msglength             int
	rawByteRead           []byte
	msgReadCh             chan []byte
	highPriorityWriteCh   chan []byte
	normalPriorityWriteCh chan []byte
	msgNotifyCh           chan bool
	quitRdCh              chan bool
	quitWrCh              chan bool
}

func NewStream(s network.Stream) *Stream {
	return &Stream{
		s.Conn().RemotePeer(),
		s.Conn().RemoteMultiaddr(),
		s,
		0,
		[]byte{},
		make(chan []byte, 100),
		make(chan []byte, highPriorityChLength),
		make(chan []byte, normalPriorityChLength),
		make(chan bool, WriteChTotalLength),
		make(chan bool, 1), //two channels to stop
		make(chan bool, 1),
	}
}

func (s *Stream) Start(quitCh chan<- *Stream, dispatch chan *streamMsg) {
	logger.Info("Stream: Start new stream")
	rw := bufio.NewReadWriter(bufio.NewReader(s.stream), bufio.NewWriter(s.stream))
	s.startLoop(rw, quitCh, dispatch)
}

func (s *Stream) StopStream(err error) {
	logger.WithFields(logger.Fields{
		"peer_address": s.remoteAddr,
		"pid":          s.peerID,
		"error":        err,
	}).Info("Stream: is terminated!!")
	s.quitRdCh <- true
	s.quitWrCh <- true
	s.stream.Close()
}

func (s *Stream) Send(data []byte, priority int) {
	defer func() {
		if p := recover(); p != nil {
			logger.WithFields(logger.Fields{
				"peer_address": s.remoteAddr,
				"pid":          s.peerID,
				"error":        p,
			}).Info("Stream: data channel closed.")
		}
	}()

	switch priority {
	case HighPriorityCommand:
		select {
		case s.highPriorityWriteCh <- data:
		default:
			logger.WithFields(logger.Fields{
				"dataCh_len": len(s.highPriorityWriteCh),
			}).Warn("Stream: High priority message channel full!")
			return
		}
	case NormalPriorityCommand:
		select {
		case s.normalPriorityWriteCh <- data:
		default:
			logger.WithFields(logger.Fields{
				"dataCh_len": len(s.normalPriorityWriteCh),
			}).Warn("Stream: normal priority message channel full!")
			return
		}
	default:
		logger.WithFields(logger.Fields{
			"priority": priority,
		}).Warn("Stream: priority is invalid!")
		return
	}

	select {
	case s.msgNotifyCh <- true:
	default:
		logger.WithFields(logger.Fields{
			"dataCh_len": len(s.msgNotifyCh),
		}).Warn("Stream: message notification channel full!")
	}

}

func (s *Stream) startLoop(rw *bufio.ReadWriter, quitCh chan<- *Stream, dispatch chan *streamMsg) {
	go s.readLoop(rw, quitCh, dispatch)
	go s.writeLoop(rw)
}

func (s *Stream) read(rw *bufio.ReadWriter) {
	buffer := make([]byte, 1024)
	var err error

	n, err := rw.Read(buffer)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"num_of_byte_read": n,
		}).Warn("Stream: Read failed")
		s.StopStream(err)
	}

	s.rawByteRead = append(s.rawByteRead, buffer[:n]...)

	for {
		if len(s.rawByteRead) < headerLength {
			return
		}

		if err = verifyHeader(s.rawByteRead[:headerLength]); err != nil {
			s.StopStream(err)
		}
		s.msglength = getLength(s.rawByteRead[:headerLength])

		if len(s.rawByteRead) < headerLength+s.msglength {
			return
		}

		s.msgReadCh <- s.rawByteRead[:headerLength+s.msglength]
		s.rawByteRead = s.rawByteRead[headerLength+s.msglength:]
	}

}

func (s *Stream) readLoop(rw *bufio.ReadWriter, quitCh chan<- *Stream, dispatch chan *streamMsg) {
	for {
		select {
		case <-s.quitRdCh:
			quitCh <- s
			logger.Debug("Stream: read loop is terminated!")
			return
		case msg := <-s.msgReadCh:
			dm := s.parseData(msg)

			dispatch <- newMsg(dm, s.peerID)
		default:
			t1 := time.Now().UnixNano()/1e6
			s.read(rw)
			cost := time.Now().UnixNano()/1e6-t1
			logger.Debugf("read cost time: %v, stream remote addr: %v, stream local addr: %v, peerId: %v",cost,s.stream.Conn().RemotePeer(), s.stream.Conn().LocalPeer(), s.peerID)
		}
	}
}

func encodeMessage(data []byte) []byte {
	header := constructHeader(data)
	ret := append(header, data...)
	return ret
}

func constructHeader(data []byte) []byte {
	length := len(data)
	msg := make([]byte, lengthByteLength)
	lengthBytes := big.NewInt(int64(length)).Bytes()
	lenDiff := len(msg) - len(lengthBytes)
	for i, b := range lengthBytes {
		msg[i+lenDiff] = b
	}
	ret := append(startBytes, msg...)
	cs := checkSum(ret)
	ret = append(ret, cs)
	return ret
}

func checkSum(data []byte) byte {
	sum := byte(0)
	for _, d := range data {
		sum += d
	}
	return sum
}

func decodeMessage(data []byte) ([]byte, error) {

	if len(data) <= headerLength {
		return nil, ErrLengthTooShort
	}

	header := data[:headerLength]
	if err := verifyHeader(header); err != nil {
		return nil, err
	}

	if len(data) != getLength(header)+headerLength {
		return nil, ErrFragmentedData
	}

	return data[headerLength:], nil
}

func verifyHeader(header []byte) error {
	if !containStartingBytes(header) {
		return ErrInvalidMessageFormat
	}

	if len(header) != headerLength {
		return ErrLengthTooShort
	}

	cs := checkSum(header[:headerLength-1])

	if cs != header[headerLength-1] {
		return ErrCheckSumIncorrect
	}
	return nil
}

func getLength(header []byte) int {
	lengthByte := header[2 : 2+lengthByteLength]
	l := *new(big.Int).SetBytes(lengthByte)
	return int(l.Uint64())
}

func containStartingBytes(data []byte) bool {
	if len(data) < len(startBytes) {
		return false
	}
	return reflect.DeepEqual(data[0:len(startBytes)], startBytes)
}

func (s *Stream) writeLoop(rw *bufio.ReadWriter) error {

	for {
		select {
		case <-s.quitWrCh:
			// Fix bug when send data to peer simultaneous with close stream,
			// and send will hang because highPriorityWriteCh is full.
			close(s.highPriorityWriteCh)
			close(s.normalPriorityWriteCh)
			close(s.msgNotifyCh)
			logger.Debug("Stream: write loop is terminated!")
			return nil
		case <-s.msgNotifyCh:
			select {
			case data := <-s.highPriorityWriteCh:
				t1 := time.Now().UnixNano()
				n, err := s.stream.Write(encodeMessage(data))
				cost := (time.Now().UnixNano()-t1)/1e6
				logger.Debugf("High priority write cost : %v, peerId: %v", cost, s.peerID)
				if err != nil {
					logger.WithError(err).WithFields(logger.Fields{
						"num_of_bytes_sent": n,
						"orig_data_size":    len(encodeMessage(data)),
					}).Warn("Stream: Send message through high priority channel failed!")
				}
				continue
			default:
			}
			select {
			case data := <-s.normalPriorityWriteCh:
				t1 := time.Now().UnixNano()
				n, err := s.stream.Write(encodeMessage(data))
				cost := (time.Now().UnixNano()-t1)/1e6
				logger.Debugf("Normal priority write cost : %v, peerId: %v", cost, s.peerID)
				if err != nil {
					logger.WithError(err).WithFields(logger.Fields{
						"num_of_bytes_sent": n,
						"orig_data_size":    len(encodeMessage(data)),
					}).Warn("Stream: Send message through normal priority channel failed!")
				}
				continue
			default:
			}
		}

	}
	return nil
}

//should parse and relay
func (s *Stream) parseData(data []byte) *DapMsg {

	dataDecoded, err := decodeMessage(data)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"data":   data,
			"length": len(data),
		}).Warn("Stream: cannot decode the message.")
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
