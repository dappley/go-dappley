package consensus

type Dynasty struct{
	miners 		[]string
}

const (
	maxProducers 		= 3
	timeBetweenBlock 	= 15
	dynastyTime 		= maxProducers*timeBetweenBlock
)

func NewDynasty() *Dynasty{return &Dynasty{}}

func NewDynastyWithMiners(miners []string) *Dynasty{return &Dynasty{miners}}

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

	dynastyTimeElapsed := int(now % dynastyTime)

	if dynastyTimeElapsed/timeBetweenBlock == minerIndex && dynastyTimeElapsed%timeBetweenBlock == 0 {
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