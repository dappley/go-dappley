package network

import (
	"context"
	"time"

	"github.com/dappley/go-dappley/common/log"
	errorValues "github.com/dappley/go-dappley/errors"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/network/networkmodel"
)

type PingResult struct {
	ID      peer.ID
	Latency *float64
}

type PingService struct {
	service  *ping.PingService
	stop     chan bool
	interval time.Duration
	started  bool
}

//NewPingService returns a new instance of PingService or an error if specified parameters are invalid
func NewPingService(host host.Host, interval time.Duration) (*PingService, error) {
	if host == nil || interval <= 0 {
		return nil, errorValues.ErrInvalidPingParamaters
	}

	return &PingService{
		service:  ping.NewPingService(host),
		stop:     make(chan bool),
		interval: interval,
		started:  false,
	}, nil
}

//Start pings peers specified by getPeers() at PingService.interval invoking a callback with a list of PingResult
func (ps *PingService) Start(getPeers func() map[peer.ID]networkmodel.PeerInfo, callback func([]*PingResult)) error {
	if !ps.started {
		go func() {
			defer log.CrashHandler()

			logger.Debug("PingService: Starting ping service...")
			ticker := time.NewTicker(ps.interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					ps.pingPeers(getPeers(), callback)
				case <-ps.stop:
					logger.Debug("PingService: Stopping ping service...")
					ps.stop <- true
					return
				}
			}
		}()
		ps.started = true
		return nil
	} else {
		return errorValues.ErrPingServiceRunning
	}
}

//Stop pinging peers
func (ps *PingService) Stop() error {
	if ps.started {
		ps.stop <- true
		<-ps.stop
		ps.started = false
		return nil
	} else {
		return errorValues.ErrPingServiceNotStarted
	}
}

func (ps *PingService) pingPeers(peers map[peer.ID]networkmodel.PeerInfo, callback func([]*PingResult)) {
	logger.Debug("PingService: pinging peers...")
	resultsCh := make(chan *PingResult)
	for _, p := range peers {
		go func(peerID peer.ID) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := <-ps.service.Ping(ctx, peerID)
			if result.Error != nil {
				logger.WithError(result.Error).Errorf("PingService: error pinging peer %v", peerID.Pretty())
				resultsCh <- &PingResult{ID: peerID, Latency: nil}
			} else {
				rtt := float64(result.RTT) / 1e6
				resultsCh <- &PingResult{ID: peerID, Latency: &rtt}
			}
		}(p.PeerId)
	}

	pingResults := make([]*PingResult, len(peers))
	for i := 0; i < len(peers); i++ {
		pingResults[i] = <-resultsCh
		logger.Debugf("PingService: received ping reply from peer: %v", pingResults[i].ID.Pretty())
	}
	callback(pingResults)
	logger.Debug("PingService: done pinging peers...")
}
