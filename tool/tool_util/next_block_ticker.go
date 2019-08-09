package tool

import (
	"time"

	"github.com/dappley/go-dappley/sdk"
	logger "github.com/sirupsen/logrus"
)

type NextBlockTicker struct {
	dappSdk *sdk.DappSdk
	ticker  chan bool
}

func NewNextBlockTicker(dappSdk *sdk.DappSdk) *NextBlockTicker {
	return &NextBlockTicker{
		dappSdk,
		make(chan bool, 1),
	}
}

func (ticker *NextBlockTicker) GetTickerChan() chan bool {
	return ticker.ticker
}

func (ticker *NextBlockTicker) Run() {
	go func() {
		timeTicker := time.NewTicker(time.Millisecond * 200).C
		currHeight := uint64(0)
		for {
			select {
			case <-timeTicker:
				height, err := ticker.dappSdk.GetBlockHeight()

				if err != nil {
					logger.Error("DappSdkNewBlockticker: Can not get block height from server")
					return
				}

				if currHeight == 0 {
					currHeight = height
				}

				if height > currHeight {
					logger.WithFields(logger.Fields{
						"height": height,
					}).Info("DappSdkNewBlockticker: New block detected!")

					if len(ticker.ticker) < cap(ticker.ticker) {
						ticker.ticker <- true
					}
					currHeight = height
				}
			}
		}
	}()
}
