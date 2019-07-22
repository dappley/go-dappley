package network_model

type PeerConnectionConfig struct {
	maxConnectionOutCount int
	maxConnectionInCount  int
}

//NewPeerConnectionConfig creates a new NewPeerConnectionConfig instance
func NewPeerConnectionConfig(maxConnectionOutCount int, maxConnectionInCount int) PeerConnectionConfig {
	return PeerConnectionConfig{
		maxConnectionOutCount,
		maxConnectionInCount,
	}
}

//GetMaxConnectionOutCount gets the maximum number of outward connections
func (config *PeerConnectionConfig) GetMaxConnectionOutCount() int {
	return config.maxConnectionOutCount
}

//GetMaxConnectionInCount gets the maximum number of inward connections
func (config *PeerConnectionConfig) GetMaxConnectionInCount() int { return config.maxConnectionInCount }

//SetMaxConnectionOutCount sets the maximum number of outward connections
func (config *PeerConnectionConfig) SetMaxConnectionOutCount(maxConnectionOutCount int) {
	config.maxConnectionOutCount = maxConnectionOutCount
}

//SetMaxConnectionInCount sets the maximum number of inward connections
func (config *PeerConnectionConfig) SetMaxConnectionInCount(maxConnectionInCount int) {
	config.maxConnectionInCount = maxConnectionInCount
}
