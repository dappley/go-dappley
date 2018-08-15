package consensus

import "github.com/dappley/go-dappley/core"

type Dynasty struct{
	producers      []string
	maxProducers   int
	timeBetweenBlk int
	dynastyTime    int
}

const (
	defaultMaxProducers   = 3
	defaultTimeBetweenBlk = 15
	defaultDynastyTime    = defaultMaxProducers * defaultTimeBetweenBlk
)

func NewDynasty() *Dynasty{
	return &Dynasty{
		producers:      []string{},
		maxProducers:   defaultMaxProducers,
		timeBetweenBlk: defaultTimeBetweenBlk,
		dynastyTime:    defaultDynastyTime,
	}
}

func NewDynastyWithProducers(producers []string) *Dynasty{
	validProducers := []string{}
	for _, producer := range producers {
		if IsProducerAddressValid(producer){
			validProducers = append(validProducers, producer)
		}
	}
	return &Dynasty{
		producers:      validProducers,
		maxProducers:   len(validProducers),
		timeBetweenBlk: defaultTimeBetweenBlk,
		dynastyTime:    len(validProducers)*defaultTimeBetweenBlk,
	}
}

func (dynasty *Dynasty) SetMaxProducers(maxProducers int){
	if maxProducers >=0 {
		dynasty.maxProducers = maxProducers
		dynasty.dynastyTime = maxProducers * dynasty.timeBetweenBlk
	}
	if maxProducers < len(dynasty.producers){
		dynasty.producers = dynasty.producers[:maxProducers]
	}
}

func (dynasty *Dynasty) SetTimeBetweenBlk(timeBetweenBlk int){
	if timeBetweenBlk > 0 {
		dynasty.timeBetweenBlk = timeBetweenBlk
		dynasty.dynastyTime = dynasty.maxProducers * timeBetweenBlk
	}
}

func (dynasty *Dynasty) AddProducer(producer string){
	if IsProducerAddressValid(producer) && len(dynasty.producers) < dynasty.maxProducers{
		dynasty.producers = append(dynasty.producers, producer)
	}
}

func (dynasty *Dynasty) AddMultipleProducers(producers []string){
	for _, producer := range producers {
		dynasty.AddProducer(producer)
	}
}

func (dynasty *Dynasty) IsMyTurn(producer string, now int64) bool{
	index := dynasty.GetProducerIndex(producer)
	return dynasty.isMyTurnByIndex(index, now)
}

func (dynasty *Dynasty) isMyTurnByIndex(producerIndex int, now int64) bool{
	if producerIndex < 0 {
		return false
	}

	dynastyTimeElapsed := int(now % int64(dynasty.dynastyTime))

	if dynastyTimeElapsed/dynasty.timeBetweenBlk == producerIndex && dynastyTimeElapsed%dynasty.timeBetweenBlk == 0 {
		return true
	}

	return false
}

//find the index of the producer. If not found, return -1
func (dynasty *Dynasty) GetProducerIndex(producer string) int{
	for i,m := range dynasty.producers {
		if producer == m {
			return i
		}
	}
	return -1
}

func IsProducerAddressValid(producer string) bool{
	addr := core.Address{producer}
	return addr.ValidateAddress()
}