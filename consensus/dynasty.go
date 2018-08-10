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

//return next mint time by unix time
func (dynasty *Dynasty) GetNextMintTime(miner string, now int64) int64{
	index := dynasty.GetMinerIndex(miner)
	return dynasty.GetNextMintTimeByIndex(index, now)
}

//return next mint time by unix time
func (dynasty *Dynasty) GetNextMintTimeByIndex(minerIndex int, now int64) int64{
	if minerIndex < 0 || minerIndex >= maxProducers{
		return -1
	}

	if now <=0 {
		return -1
	}

	dynastyTimeElapsed := now % dynastyTime
	dynastyBeginTime := now - dynastyTimeElapsed

	if int(dynastyTimeElapsed) < timeBetweenBlock * minerIndex{
		return dynastyBeginTime + int64(timeBetweenBlock * minerIndex)
	}else{
		return dynastyBeginTime + int64(timeBetweenBlock * minerIndex) + dynastyTime
	}
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