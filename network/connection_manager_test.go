package network

import (
	"github.com/dappley/go-dappley/network/networkmodel"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConnectionManager_AddConnection(t *testing.T) {
	tests := []struct {
		name                  string
		connectionType        ConnectionType
		expectedConnectionIn  int
		expectedConnectionOut int
	}{
		{
			name:                  "connection in",
			connectionType:        ConnectionTypeIn,
			expectedConnectionIn:  1,
			expectedConnectionOut: 0,
		},
		{
			name:                  "connection 0ut",
			connectionType:        ConnectionTypeOut,
			expectedConnectionIn:  0,
			expectedConnectionOut: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewConnectionManager(networkmodel.NewPeerConnectionConfig(2, 2))
			m.AddConnection(tt.connectionType)
			assert.Equal(t, tt.expectedConnectionIn, m.connectionInCount)
			assert.Equal(t, tt.expectedConnectionOut, m.connectionOutCount)
		})
	}
}

func TestConnectionManager_RemoveConnection(t *testing.T) {
	tests := []struct {
		name                  string
		connectionType        ConnectionType
		startingConnectionIn  int
		startingConnectionOut int
		expectedConnectionIn  int
		expectedConnectionOut int
	}{
		{
			name:                  "connection in",
			connectionType:        ConnectionTypeIn,
			startingConnectionIn:  1,
			startingConnectionOut: 1,
			expectedConnectionIn:  0,
			expectedConnectionOut: 1,
		},
		{
			name:                  "connection out",
			connectionType:        ConnectionTypeOut,
			startingConnectionIn:  1,
			startingConnectionOut: 1,
			expectedConnectionIn:  1,
			expectedConnectionOut: 0,
		},
		{
			name:                  "0 connection in",
			connectionType:        ConnectionTypeIn,
			startingConnectionIn:  0,
			startingConnectionOut: 0,
			expectedConnectionIn:  0,
			expectedConnectionOut: 0,
		},
		{
			name:                  "0 connection out",
			connectionType:        ConnectionTypeOut,
			startingConnectionIn:  0,
			startingConnectionOut: 0,
			expectedConnectionIn:  0,
			expectedConnectionOut: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewConnectionManager(networkmodel.NewPeerConnectionConfig(2, 2))
			m.connectionInCount = tt.startingConnectionIn
			m.connectionOutCount = tt.startingConnectionOut
			m.RemoveConnection(tt.connectionType)
			assert.Equal(t, tt.expectedConnectionIn, m.connectionInCount)
			assert.Equal(t, tt.expectedConnectionOut, m.connectionOutCount)
		})
	}
}

func TestConnectionManager_IsConnectionFull(t *testing.T) {
	tests := []struct {
		name                  string
		connectionType        ConnectionType
		maxConnectionIn       int
		maxConnectionOut      int
		startingConnectionIn  int
		startingConnectionOut int
		expectedResult        bool
	}{
		{
			name:                  "connection in full/input: connectionIn",
			connectionType:        ConnectionTypeIn,
			maxConnectionIn:       2,
			maxConnectionOut:      2,
			startingConnectionIn:  2,
			startingConnectionOut: 1,
			expectedResult:        true,
		},
		{
			name:                  "connection in full/input: connectionOut",
			connectionType:        ConnectionTypeOut,
			maxConnectionIn:       2,
			maxConnectionOut:      2,
			startingConnectionIn:  2,
			startingConnectionOut: 1,
			expectedResult:        false,
		},
		{
			name:                  "connection out full/input: connectionIn",
			connectionType:        ConnectionTypeIn,
			maxConnectionIn:       2,
			maxConnectionOut:      2,
			startingConnectionIn:  1,
			startingConnectionOut: 2,
			expectedResult:        false,
		},
		{
			name:                  "connection out full/input: connectionOut",
			connectionType:        ConnectionTypeOut,
			maxConnectionIn:       2,
			maxConnectionOut:      2,
			startingConnectionIn:  1,
			startingConnectionOut: 2,
			expectedResult:        true,
		},
		{
			name:                  "connection not full/input: connectionIn",
			connectionType:        ConnectionTypeIn,
			maxConnectionIn:       2,
			maxConnectionOut:      2,
			startingConnectionIn:  1,
			startingConnectionOut: 1,
			expectedResult:        false,
		},
		{
			name:                  "connection not full/input: connectionOut",
			connectionType:        ConnectionTypeOut,
			maxConnectionIn:       2,
			maxConnectionOut:      2,
			startingConnectionIn:  1,
			startingConnectionOut: 1,
			expectedResult:        false,
		},
		{
			name:                  "connection full/input: connectionIn",
			connectionType:        ConnectionTypeIn,
			maxConnectionIn:       2,
			maxConnectionOut:      2,
			startingConnectionIn:  2,
			startingConnectionOut: 2,
			expectedResult:        true,
		},
		{
			name:                  "connection full/input: connectionOut",
			connectionType:        ConnectionTypeOut,
			maxConnectionIn:       2,
			maxConnectionOut:      2,
			startingConnectionIn:  2,
			startingConnectionOut: 2,
			expectedResult:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewConnectionManager(networkmodel.NewPeerConnectionConfig(tt.maxConnectionOut, tt.maxConnectionIn))
			m.connectionInCount = tt.startingConnectionIn
			m.connectionOutCount = tt.startingConnectionOut
			assert.Equal(t, tt.expectedResult, m.IsConnectionFull(tt.connectionType))
		})
	}
}

func TestConnectionManager_GetNumOfConnectionsAllowed(t *testing.T) {
	tests := []struct {
		name                        string
		maxConnectionIn             int
		maxConnectionOut            int
		startingConnectionIn        int
		startingConnectionOut       int
		expectedConnectionInResult  int
		expectedConnectionOutResult int
	}{
		{
			name:                        "normal case",
			maxConnectionIn:             2,
			maxConnectionOut:            2,
			startingConnectionIn:        1,
			startingConnectionOut:       1,
			expectedConnectionInResult:  1,
			expectedConnectionOutResult: 1,
		},
		{
			name:                        "connection full",
			maxConnectionIn:             2,
			maxConnectionOut:            2,
			startingConnectionIn:        2,
			startingConnectionOut:       2,
			expectedConnectionInResult:  0,
			expectedConnectionOutResult: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewConnectionManager(networkmodel.NewPeerConnectionConfig(tt.maxConnectionOut, tt.maxConnectionIn))
			m.connectionInCount = tt.startingConnectionIn
			m.connectionOutCount = tt.startingConnectionOut
			assert.Equal(t, tt.expectedConnectionInResult, m.GetNumOfConnectionsAllowed(ConnectionTypeIn))
			assert.Equal(t, tt.expectedConnectionOutResult, m.GetNumOfConnectionsAllowed(ConnectionTypeOut))
		})
	}
}
