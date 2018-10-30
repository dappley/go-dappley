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

package consensus

import (
	"errors"
	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

type Dynasty struct {
	producers      []string
	maxProducers   int
	timeBetweenBlk int
	dynastyTime    int
}

const (
	defaultMaxProducers   = 5
	defaultTimeBetweenBlk = 15
)

func (d *Dynasty) trimProducers() {
	//if producer conf file does not have all producers
	if len(d.producers) < defaultMaxProducers {
		for len(d.producers) < defaultMaxProducers {
			d.producers = append(d.producers, "")
		}
	}
	//if producer conf file has too many producers
	if len(d.producers) > defaultMaxProducers {
		d.producers = d.producers[:defaultMaxProducers]
	}
}

func NewDynasty(producers []string, maxProducers, timeBetweenBlk int) *Dynasty {
	return &Dynasty{
		producers:      producers,
		maxProducers:   maxProducers,
		timeBetweenBlk: timeBetweenBlk,
		dynastyTime:    timeBetweenBlk * maxProducers,
	}
}

//New dynasty from config file
func NewDynastyWithConfigProducers(producers []string, maxProducers int) *Dynasty {
	validProducers := []string{}
	for _, producer := range producers {
		if IsProducerAddressValid(producer) {
			validProducers = append(validProducers, producer)
		}
	}

	if maxProducers == 0 {
		maxProducers = defaultMaxProducers
	}

	d := &Dynasty{
		producers:      validProducers,
		maxProducers:   maxProducers,
		timeBetweenBlk: defaultTimeBetweenBlk,
		dynastyTime:    maxProducers * defaultTimeBetweenBlk,
	}
	d.trimProducers()
	return d
}

func (dynasty *Dynasty) SetMaxProducers(maxProducers int) {
	if maxProducers >= 0 {
		dynasty.maxProducers = maxProducers
		dynasty.dynastyTime = maxProducers * dynasty.timeBetweenBlk
	}
	if maxProducers < len(dynasty.producers) {
		dynasty.producers = dynasty.producers[:maxProducers]
	}
}

func (dynasty *Dynasty) SetTimeBetweenBlk(timeBetweenBlk int) {
	if timeBetweenBlk > 0 {
		dynasty.timeBetweenBlk = timeBetweenBlk
		dynasty.dynastyTime = dynasty.maxProducers * timeBetweenBlk
	}
}

func (dynasty *Dynasty) AddProducer(producer string) error {
	for _, producerNow := range dynasty.producers {
		if producerNow == producer {
			return errors.New("Producer already in the producer list！")
		}
	}

	if IsProducerAddressValid(producer) && len(dynasty.producers) < dynasty.maxProducers {
		dynasty.producers = append(dynasty.producers, producer)
		logger.Debug("Current Producers:")
		for _, producerIt := range dynasty.producers {
			logger.Debug(producerIt)
		}
		return nil
	} else {
		if !IsProducerAddressValid(producer) {
			return errors.New("The address of producers not valid！")
		} else {
			return errors.New("The number of producers reaches the maximum！")
		}
	}
}

func (dynasty *Dynasty) GetProducers() []string {
	return dynasty.producers
}

func (dynasty *Dynasty) AddMultipleProducers(producers []string) {
	for _, producer := range producers {
		dynasty.AddProducer(producer)
	}
}

func (dynasty *Dynasty) IsMyTurn(producer string, now int64) bool {
	index := dynasty.GetProducerIndex(producer)
	return dynasty.isMyTurnByIndex(index, now)
}

func (dynasty *Dynasty) isMyTurnByIndex(producerIndex int, now int64) bool {
	if producerIndex < 0 {
		return false
	}
	dynastyTimeElapsed := int(now % int64(dynasty.dynastyTime))
	return dynastyTimeElapsed == producerIndex*dynasty.timeBetweenBlk
}

func (dynasty *Dynasty) ProducerAtATime(time int64) string {
	if time < 0 {
		return ""
	}
	dynastyTimeElapsed := int(time % int64(dynasty.dynastyTime))
	index := dynastyTimeElapsed / dynasty.timeBetweenBlk
	return dynasty.producers[index]
}

//find the index of the producer. If not found, return -1
func (dynasty *Dynasty) GetProducerIndex(producer string) int {
	for i, m := range dynasty.producers {
		if producer == m {
			return i
		}
	}
	return -1
}

func IsProducerAddressValid(producer string) bool {
	addr := core.NewAddress(producer)
	return addr.ValidateAddress()
}

func (dynasty *Dynasty) GetDynastyTime() int {
	return dynasty.dynastyTime
}
