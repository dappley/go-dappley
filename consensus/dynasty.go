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
	"fmt"

	"github.com/dappley/go-dappley/core/account"
	logger "github.com/sirupsen/logrus"
)

type Dynasty struct {
	producers      []string
	maxProducers   int
	timeBetweenBlk int
	dynastyTime    int
}

type DynastyReplacement struct {
	original string
	new      string
	height   uint64
	kind     int
}

const (
	defaultMaxProducers   = 21
	defaultTimeBetweenBlk = 5
)

//NewDynasty returns a new dynasty instance
func NewDynasty(producers []string, maxProducers, timeBetweenBlk int) *Dynasty {
	return &Dynasty{
		producers:      producers,
		maxProducers:   maxProducers,
		timeBetweenBlk: timeBetweenBlk,
		dynastyTime:    timeBetweenBlk * maxProducers,
	}
}

func NewDynastyReplacement(original, new string, height uint64, kind int) *DynastyReplacement {
	return &DynastyReplacement{
		original: original,
		new:      new,
		height:   height,
		kind:     kind,
	}
}

//NewDynastyWithConfigProducers returns a new dynasty from config file
func NewDynastyWithConfigProducers(producers []string, maxProducers int) *Dynasty {
	validProducers := []string{}
	for _, producer := range producers {
		producerAccount := account.NewTransactionAccountByAddress(account.NewAddress(producer))
		if producerAccount.IsValid() {
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

//trimProducers deletes producers if the number of producers are more than the maximum
func (dynasty *Dynasty) trimProducers() {
	//if producer conf file has too many producers
	if len(dynasty.producers) > defaultMaxProducers {
		dynasty.producers = dynasty.producers[:defaultMaxProducers]
	}
}

//GetMaxProducers returns the maximum number of producers allowed in the dynasty
func (dynasty *Dynasty) GetMaxProducers() int {
	return dynasty.maxProducers
}

//SetMaxProducers sets the maximum number of producers allowed in the dynasty
func (dynasty *Dynasty) SetMaxProducers(maxProducers int) {
	if maxProducers >= 0 {
		dynasty.maxProducers = maxProducers
		dynasty.dynastyTime = maxProducers * dynasty.timeBetweenBlk
	}
	if maxProducers < len(dynasty.producers) {
		dynasty.producers = dynasty.producers[:maxProducers]
	}
}

//SetTimeBetweenBlk sets the block time
func (dynasty *Dynasty) SetTimeBetweenBlk(timeBetweenBlk int) {
	if timeBetweenBlk > 0 {
		dynasty.timeBetweenBlk = timeBetweenBlk
		dynasty.dynastyTime = dynasty.maxProducers * timeBetweenBlk
	}
}

//AddProducer adds a producer to the dynasty
func (dynasty *Dynasty) AddProducer(producer string) error {
	if err := dynasty.isAddingProducerAllowed(producer); err != nil {
		return err
	}
	dynasty.producers = append(dynasty.producers, producer)
	logger.WithFields(logger.Fields{
		"producer": producer,
		"list":     dynasty.producers,
	}).Debug("Dynasty: added a producer to list.")
	return nil
}

//isAddingProducerAllowed checks if adding a producer is allowed
func (dynasty *Dynasty) isAddingProducerAllowed(producer string) error {
	for _, producerNow := range dynasty.producers {
		if producerNow == producer {
			return errors.New("already a producer")
		}
	}
	producerAccount := account.NewTransactionAccountByAddress(account.NewAddress(producer))

	if producerAccount.IsValid() && len(dynasty.producers) < dynasty.maxProducers {
		return nil
	}

	if !producerAccount.IsValid() {
		return errors.New("invalid producer address")
	}
	return errors.New("maximum number of producers reached")
}

//GetProducers returns all producers
func (dynasty *Dynasty) GetProducers() []string {
	return dynasty.producers
}

//AddMultipleProducers adds multipled producers to the dynasty
func (dynasty *Dynasty) AddMultipleProducers(producers []string) {
	for _, producer := range producers {
		dynasty.AddProducer(producer)
	}
}

//IsMyTurn returns if it is the input producer's turn to produce block
func (dynasty *Dynasty) IsMyTurn(producer string, now int64) bool {
	index := dynasty.GetProducerIndex(producer)
	return dynasty.isMyTurnByIndex(index, now)
}

//isMyTurnByIndex returns if it is the turn for the producer with producerIndex to produce block
func (dynasty *Dynasty) isMyTurnByIndex(producerIndex int, now int64) bool {
	if producerIndex < 0 {
		return false
	}
	dynastyTimeElapsed := int(now % int64(dynasty.dynastyTime))
	return dynastyTimeElapsed == producerIndex*dynasty.timeBetweenBlk
}

//ProducerAtATime returns the expected producer at the input time
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

//GetDynastyTime returns the dynasty time
func (dynasty *Dynasty) GetDynastyTime() int {
	return dynasty.dynastyTime
}

//IsSettingProducersAllowed returns if setting producers is allowed
func (dynasty *Dynasty) IsSettingProducersAllowed(producers []string, maxProducers ...int) error {

	maxProd := dynasty.maxProducers
	if len(maxProducers) > 0 {
		maxProd = maxProducers[0]
	}

	if len(producers) > maxProd {
		return errors.New("can not exceed maximum number of producers")
	}

	seen := make(map[string]bool)
	for _, producer := range producers {
		producerAccount := account.NewTransactionAccountByAddress(account.NewAddress(producer))
		if seen[producer] {
			return errors.New(fmt.Sprintf("can not add a duplicate producer: \"%v\"", producer))
		}

		if !producerAccount.IsValid() {
			return errors.New(fmt.Sprintf("\"%v\" is a invalid producer", producer))
		}
		seen[producer] = true
	}

	return nil
}

//SetProducers sets the producers
func (dynasty *Dynasty) SetProducers(producers []string) {
	dynasty.producers = producers
}
