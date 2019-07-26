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
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
)

const (
	highPriorityChLength   = 1024 * 4
	normalPriorityChLength = 1024 * 4
	WriteChTotalLength     = highPriorityChLength + normalPriorityChLength
)

type Stream struct {
	peerInfo              network_model.PeerInfo
	stream                network.Stream
	rawByteRead           []byte
	highPriorityWriteCh   chan *network_model.DappPacket
	normalPriorityWriteCh chan *network_model.DappPacket
	msgNotifyCh           chan bool
	quitRdCh              chan bool
	quitWrCh              chan bool
}

//NewStream creates a new Stream instance
func NewStream(s network.Stream) *Stream {
	return &Stream{
		network_model.PeerInfo{s.Conn().RemotePeer(), []multiaddr.Multiaddr{s.Conn().RemoteMultiaddr()}},
		s,
		[]byte{},
		make(chan *network_model.DappPacket, highPriorityChLength),
		make(chan *network_model.DappPacket, normalPriorityChLength),
		make(chan bool, WriteChTotalLength),
		make(chan bool, 1), //two channels to stop
		make(chan bool, 1),
	}
}

//GetPeerId returns the remote peer ID
func (s *Stream) GetPeerId() peer.ID { return s.peerInfo.PeerId }

//GetRemoteAddr returns the remote multi address
func (s *Stream) GetRemoteAddr() multiaddr.Multiaddr { return s.peerInfo.Addrs[0] }

//Start starts a stream with a peer
func (s *Stream) Start(quitCh chan<- *Stream, msgRcvCh chan *network_model.DappPacketContext) {
	logger.Debug("Stream: Start new stream")
	rw := bufio.NewReadWriter(bufio.NewReader(s.stream), bufio.NewWriter(s.stream))
	s.startLoop(rw, quitCh, msgRcvCh)
}

//StopStream stops a stream
func (s *Stream) StopStream() {
	logger.WithFields(logger.Fields{
		"peer_address": s.GetRemoteAddr(),
		"pid":          s.GetPeerId(),
	}).Info("Stream: stream is terminated")
	s.quitRdCh <- true
	s.quitWrCh <- true
	s.stream.Close()
}

//Send sends a DappPacket to its peer
func (s *Stream) Send(packet *network_model.DappPacket, priority network_model.DappCmdPriority) {
	defer func() {
		if p := recover(); p != nil {
			logger.WithFields(logger.Fields{
				"peer_address": s.GetRemoteAddr(),
				"pid":          s.GetPeerId(),
				"error":        p,
			}).Info("Stream: data channel closed.")
		}
	}()

	switch priority {
	case network_model.HighPriorityCommand:
		select {
		case s.highPriorityWriteCh <- packet:
		default:
			logger.WithFields(logger.Fields{
				"dataCh_len": len(s.highPriorityWriteCh),
			}).Warn("Stream: High priority message channel full!")
			return
		}
	case network_model.NormalPriorityCommand:
		select {
		case s.normalPriorityWriteCh <- packet:
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

//startLoop starts the read and write loop
func (s *Stream) startLoop(rw *bufio.ReadWriter, quitCh chan<- *Stream, msgRcvCh chan *network_model.DappPacketContext) {
	go s.readLoop(rw, quitCh, msgRcvCh)
	go s.writeLoop(rw)
}

//read reads raw bytes from its peer
func (s *Stream) read(rw *bufio.ReadWriter, msgRcvCh chan *network_model.DappPacketContext) {
	buffer := make([]byte, 1024)
	var err error

	n, err := rw.Read(buffer)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"num_of_byte_read": n,
		}).Warn("Stream: Read failed")
		s.StopStream()
	}

	s.rawByteRead = append(s.rawByteRead, buffer[:n]...)

	for {
		packet, err := network_model.DeserializeIntoDappPacket(s.rawByteRead)

		if err != nil {
			if err == network_model.ErrLengthTooShort {
				return
			} else {
				logger.WithError(err).WithFields(logger.Fields{
					"num_of_byte_read": n,
				}).Warn("Stream: Parse packet failed")
				s.StopStream()
			}
		}

		select {
		case msgRcvCh <- &network_model.DappPacketContext{packet, s.GetPeerId()}:
		default:
			logger.WithFields(logger.Fields{
				"dispatchCh_len": len(msgRcvCh),
			}).Warn("Stream: message receive channel full!")
			return
		}
		s.rawByteRead = s.rawByteRead[packet.GetLength():]
	}

}

//readLoop keeps reading from its peer
func (s *Stream) readLoop(rw *bufio.ReadWriter, quitCh chan<- *Stream, msgRcvCh chan *network_model.DappPacketContext) {
	for {
		select {
		case <-s.quitRdCh:
			quitCh <- s
			logger.Debug("Stream: read loop is terminated!")
			return
		default:
			s.read(rw, msgRcvCh)
		}
	}
}

//writeLoop listens to all write channels and sends the message to its peer
func (s *Stream) writeLoop(rw *bufio.ReadWriter) error {

	for {
		select {
		case <-s.quitWrCh:
			// Fix bug when send packet to peer simultaneous with close stream,
			// and send will hang because highPriorityWriteCh is full.
			close(s.highPriorityWriteCh)
			close(s.normalPriorityWriteCh)
			close(s.msgNotifyCh)
			logger.Debug("Stream: write loop is terminated!")
			return nil
		case <-s.msgNotifyCh:
			select {
			case packet := <-s.highPriorityWriteCh:
				n, err := s.stream.Write(packet.GetRawBytes())
				if err != nil {
					logger.WithError(err).WithFields(logger.Fields{
						"num_of_bytes_sent": n,
						"orig_data_size":    packet.GetLength(),
					}).Warn("Stream: Send message through high priority channel failed!")
				}
				continue
			default:
			}
			select {
			case packet := <-s.normalPriorityWriteCh:
				n, err := s.stream.Write(packet.GetRawBytes())
				if err != nil {
					logger.WithError(err).WithFields(logger.Fields{
						"num_of_bytes_sent": n,
						"orig_data_size":    packet.GetLength(),
					}).Warn("Stream: Send message through normal priority channel failed!")
				}
				continue
			default:
			}
		}

	}
	return nil
}
