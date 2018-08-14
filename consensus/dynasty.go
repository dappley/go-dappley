package consensus

type Dynasty struct{
	miners 			[]string
	maxProducers 	int
	timeBetweenBlk 	int
	dynastyTime 	int
}

const (
	defaultMaxProducers     = 3
	defaultTimeBetweenBlock = 15
	defaultDynastyTime      = defaultMaxProducers * defaultTimeBetweenBlock
)

func NewDynasty() *Dynasty{
	return &Dynasty{
		maxProducers:   defaultMaxProducers,
		timeBetweenBlk: defaultTimeBetweenBlock,
		dynastyTime:    defaultDynastyTime,
	}
}

func NewDynastyWithMiners(miners []string) *Dynasty{
	return &Dynasty{
		miners:         miners,
		maxProducers:   defaultMaxProducers,
		timeBetweenBlk: defaultTimeBetweenBlock,
		dynastyTime:    defaultDynastyTime,
	}
}

func (dynasty *Dynasty) SetMaxProducers(maxProducers int){
	dynasty.maxProducers = maxProducers
	dynasty.dynastyTime = maxProducers * dynasty.timeBetweenBlk
}

func (dynasty *Dynasty) SetTimeBetweenBlk(timeBetweenBlk int){
	dynasty.timeBetweenBlk = timeBetweenBlk
	dynasty.dynastyTime = dynasty.maxProducers * timeBetweenBlk
}

func (dynasty *Dynasty) AddMiner(miner string){
	dynasty.miners = append(dynasty.miners, miner)
}

func (dynasty *Dynasty) AddMultipleMiners(miners []string){
	dynasty.miners = append(dynasty.miners, miners...)
}

func (dynasty *Dynasty) IsMyTurn(miner string, now int64) bool{
	index := dynasty.GetMinerIndex(miner)
	return dynasty.isMyTurnByIndex(index, now)
}

func (dynasty *Dynasty) isMyTurnByIndex(minerIndex int, now int64) bool{
	if minerIndex < 0 {
		return false
	}

	dynastyTimeElapsed := int(now % int64(dynasty.dynastyTime))

	if dynastyTimeElapsed/dynasty.timeBetweenBlk == minerIndex && dynastyTimeElapsed%dynasty.timeBetweenBlk == 0 {
		return true
	}

	return false
}

//find the index of the miner. If not found, return -1
func (dynasty *Dynasty) GetMinerIndex(miner string) int{
	for i,m := range dynasty.miners {
		if miner == m {
			return i
		}
	}
	return -1
}