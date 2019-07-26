package network

import (
	"github.com/dappley/go-dappley/network/network_model"
	logger "github.com/sirupsen/logrus"
)

const (
	ConnectionTypeSeed ConnectionType = 0
	ConnectionTypeIn   ConnectionType = 1
	ConnectionTypeOut  ConnectionType = 2

	defaultMaxConnectionOutCount = 16
	defaultMaxConnectionInCount  = 128
)

type ConnectionManager struct {
	connectionConfig   network_model.PeerConnectionConfig
	connectionOutCount int //Connection that current node connect to other nodes, exclude seed nodes
	connectionInCount  int //Connection that other node connectionManager to current node.
}

func NewConnectionManager(config network_model.PeerConnectionConfig) *ConnectionManager {
	if config.GetMaxConnectionOutCount() == 0 {
		config.SetMaxConnectionOutCount(defaultMaxConnectionOutCount)
	}

	if config.GetMaxConnectionInCount() == 0 {
		config.SetMaxConnectionInCount(defaultMaxConnectionInCount)
	}

	return &ConnectionManager{
		config,
		0,
		0,
	}
}

func (cm *ConnectionManager) GetNumOfConnectionsAllowed(connectionType ConnectionType) int {

	result := 0

	switch connectionType {
	case ConnectionTypeIn:
		result = cm.connectionConfig.GetMaxConnectionInCount() - cm.connectionInCount
	case ConnectionTypeOut:
		result = cm.connectionConfig.GetMaxConnectionOutCount() - cm.connectionOutCount
	}

	if result < 0 {
		result = 0
	}

	return result
}

func (cm *ConnectionManager) IsConnectionFull(connectionType ConnectionType) bool {
	switch connectionType {
	case ConnectionTypeIn:
		if cm.connectionInCount >= cm.connectionConfig.GetMaxConnectionInCount() {
			logger.WithFields(logger.Fields{
				"numOfConnections": cm.connectionInCount,
				"maxConnections":   cm.connectionConfig.GetMaxConnectionInCount(),
			}).Info("ConnectionManager: inward connections have reached limit")
			return true
		}
	case ConnectionTypeOut:
		if cm.connectionOutCount >= cm.connectionConfig.GetMaxConnectionOutCount() {
			logger.WithFields(logger.Fields{
				"numOfConnections": cm.connectionOutCount,
				"maxConnections":   cm.connectionConfig.GetMaxConnectionOutCount(),
			}).Info("ConnectionManager: outward connections have reached limit")
			return true
		}
	}
	return false
}

func (cm *ConnectionManager) RemoveConnection(connectionType ConnectionType) {
	switch connectionType {
	case ConnectionTypeIn:
		cm.connectionInCount--

	case ConnectionTypeOut:
		cm.connectionOutCount--
	}
}

func (cm *ConnectionManager) AddConnection(connectionType ConnectionType) {
	switch connectionType {
	case ConnectionTypeIn:
		cm.connectionInCount++

	case ConnectionTypeOut:
		cm.connectionOutCount++
	}
}
