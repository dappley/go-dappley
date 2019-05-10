package sdk

import (
	logger "github.com/sirupsen/logrus"
	"time"
)

type DappSdkNextBlockTicker struct {
	dappSdk *DappSdk
	ticker  chan bool
}

func NewDappSdkNextBlockTicker(dappSdk *DappSdk) *DappSdkNextBlockTicker {
	return &DappSdkNextBlockTicker{
		dappSdk,
		make(chan bool, 1),
	}
}

func (sdkTicker *DappSdkNextBlockTicker) GetTickerChan() chan bool {
	return sdkTicker.ticker
}

func (sdkTicker *DappSdkNextBlockTicker) Run() {
	go func() {
		timeTicker := time.NewTicker(time.Millisecond * 200).C
		currHeight := uint64(0)
		for {
			select {
			case <-timeTicker:
				height, err := sdkTicker.dappSdk.GetBlockHeight()

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

					if len(sdkTicker.ticker) < cap(sdkTicker.ticker) {
						sdkTicker.ticker <- true
					}
					currHeight = height
				}
			}
		}
	}()
}
