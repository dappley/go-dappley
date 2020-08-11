package blockchain

type BlockchainState int

const (
	BlockchainInit BlockchainState = iota
	BlockchainDownloading
	BlockchainSync
	BlockchainReady
)

var (
	// all blockchain instances hold one state
	state BlockchainState
)

// Set state of the blockchain
func setBlockchainState(newState BlockchainState) {
	state = newState
}

// Returns current state of the blockchain
func getBlockchainState() BlockchainState {
	return state
}
