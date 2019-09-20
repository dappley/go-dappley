package network

import (
	"github.com/dappley/go-dappley/network/networkmodel"
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
	connectionConfig   networkmodel.PeerConnectionConfig
	connectionOutCount int //Connection that current node connect to other nodes, exclude seed nodes
	connectionInCount  int //Connection that other node connectionManager to current node.
}

func NewConnectionManager(config networkmodel.PeerConnectionConfig) *ConnectionManager {
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

func (cm *ConnectionManager) GetMaxConnectionInCount() int {
	return cm.connectionConfig.GetMaxConnectionInCount()
}

func (cm *ConnectionManager) GetMaxConnectionOutCount() int {
	return cm.connectionConfig.GetMaxConnectionOutCount()
}

func (cm *ConnectionManager) SetMaxConnectionInCount(maxConnectionInCount int) {
	cm.connectionConfig.SetMaxConnectionInCount(maxConnectionInCount)
}

func (cm *ConnectionManager) SetMaxConnectionOutCount(maxConnectionOutCount int) {
	cm.connectionConfig.SetMaxConnectionOutCount(maxConnectionOutCount)
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
		if cm.connectionInCount > 0 {
			cm.connectionInCount--
			ConnectionTypeInNum.Update(int64(cm.connectionInCount))
		}

	case ConnectionTypeOut:
		if cm.connectionOutCount > 0 {
			cm.connectionOutCount--
			ConnectionTypeOutNum.Update(int64(cm.connectionOutCount))
		}
	}
}

func (cm *ConnectionManager) AddConnection(connectionType ConnectionType) {
	switch connectionType {
	case ConnectionTypeIn:
		cm.connectionInCount++
		ConnectionTypeInNum.Update(int64(cm.connectionInCount))

	case ConnectionTypeOut:
		cm.connectionOutCount++
		ConnectionTypeOutNum.Update(int64(cm.connectionOutCount))
	}
}
