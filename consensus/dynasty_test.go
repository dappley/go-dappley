package consensus

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestDynasty_NewDynasty(t *testing.T) {
	dynasty := NewDynasty()
	assert.Empty(t,dynasty.miners)
}

func TestDynasty_NewDynastyWithMiners(t *testing.T) {
	tests := []struct{
		name 		string
		input 		[]string
		expected	[]string
	}{
		{
			name: 		"ValidInput",
			input:		[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			expected:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
		},
		{
			name: 		"InvalidInput",
			input:		[]string{"m1","m2","m3"},
			expected:	[]string{},
		},
		{
			name: 		"mixedInput",
			input:		[]string{"m1","121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD","m3"},
			expected:	[]string{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"},
		},
		{
			name: 		"EmptyInput",
			input:		[]string{},
			expected:	[]string{},
		},
	}

	for _,tt := range tests {
		t.Run(tt.name, func(t *testing.T){
			dynasty:= NewDynastyWithMiners(tt.input)
			assert.Equal(t, tt.expected, dynasty.miners)
		})
	}
}

func TestDynasty_AddMiner(t *testing.T) {
	tests := []struct{
		name 		string
		maxPeers    int
		input 		string
		expected	[]string
	}{
		{
			name: 		"ValidInput",
			maxPeers:   3,
			input:		"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
			expected:	[]string{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"},
		},
		{
			name: 		"MinerExceedsLimit",
			maxPeers:   0,
			input:		"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
			expected:	[]string{},
		},
		{
			name: 		"InvalidInput",
			maxPeers:   3,
			input:		"m1",
			expected:	[]string{},
		},
		{
			name: 		"EmptyInput",
			maxPeers:   3,
			input:		"",
			expected:	[]string{},
		},
	}

	for _,tt := range tests {
		t.Run(tt.name, func(t *testing.T){
			dynasty:= NewDynasty()
			dynasty.SetMaxProducers(tt.maxPeers)
			dynasty.AddMiner(tt.input)
			assert.Equal(t, tt.expected, dynasty.miners)
		})
	}
}

func TestDynasty_AddMultipleMiners(t *testing.T) {
	tests := []struct{
		name 		string
		maxPeers    int
		input 		[]string
		expected	[]string
	}{
		{
			name: 		"ValidInput",
			maxPeers:   3,
			input:		[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			expected:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
		},
		{
			name: 		"ExceedsLimit",
			maxPeers:   2,
			input:		[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			expected:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				},
		},
		{
			name: 		"InvalidInput",
			maxPeers:   3,
			input:		[]string{"m1","m2","m3"},
			expected:	[]string{},
		},
		{
			name: 		"mixedInput",
			maxPeers:   3,
			input:		[]string{"m1","121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD","m3"},
			expected:	[]string{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"},
		},
		{
			name: 		"EmptyInput",
			maxPeers:   3,
			input:		[]string{},
			expected:	[]string{},
		},
	}

	for _,tt := range tests {
		t.Run(tt.name, func(t *testing.T){
			dynasty:= NewDynasty()
			dynasty.SetMaxProducers(tt.maxPeers)
			dynasty.AddMultipleMiners(tt.input)
			assert.Equal(t, tt.expected, dynasty.miners)
		})
	}
}

func TestDynasty_GetMinerIndex(t *testing.T) {
	tests := []struct{
		name 			string
		initialMiners 	[]string
		miner 			string
		expected		int
	}{
		{
			name: 			"minerCouldBeFound",
			initialMiners:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			miner: 			"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
			expected:		0,
		},
		{
			name: 			"minerCouldNotBeFound",
			initialMiners:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			miner: 			"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDF",
			expected:		-1,
		},
		{
			name: 			"EmptyInput",
			initialMiners:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			miner: 			"",
			expected:		-1,
		},
	}

	for _,tt := range tests {
		t.Run(tt.name, func(t *testing.T){
			dynasty:= NewDynastyWithMiners(tt.initialMiners)
			index := dynasty.GetMinerIndex(tt.miner)
			assert.Equal(t, tt.expected, index)
		})
	}
}

func TestDynasty_IsMyTurnByIndex(t *testing.T) {
	tests := []struct{
		name 		string
		index 		int
		now 		int64
		expected	bool
	}{
		{
			name: 		"isMyTurn",
			index:		2,
			now: 		75,
			expected:	true,
		},
		{
			name: 		"NotMyTurn",
			index:		1,
			now: 		61,
			expected:	false,
		},
		{
			name: 		"InvalidIndexInput",
			index:		-6,
			now: 		61,
			expected:	false,
		},
		{
			name: 		"InvalidNowInput",
			index:		2,
			now: 		-1,
			expected:	false,
		},
		{
			name: 		"IndexInputExceedsMaxSize",
			index:		5,
			now: 		44,
			expected:	false,
		},
	}

	for _,tt := range tests {
		t.Run(tt.name, func(t *testing.T){
			dynasty:= NewDynasty()
			nextMintTime := dynasty.isMyTurnByIndex(tt.index, tt.now)
			assert.Equal(t, tt.expected, nextMintTime)
		})
	}
}

func TestDynasty_IsMyTurn(t *testing.T) {
	tests := []struct{
		name 			string
		initialMiners 	[]string
		miner 			string
		index 			int
		now 			int64
		expected		bool
	}{
		{
			name: 			"IsMyTurn",
			initialMiners:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			miner: 			"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct",
			now: 			75,
			expected:		true,
		},
		{
			name: 			"NotMyTurn",
			initialMiners:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			miner: 			"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
			now: 			61,
			expected:		false,
		},
		{
			name: 			"EmptyInput",
			initialMiners:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			miner: 			"",
			now: 			61,
			expected:		false,
		},
		{
			name: 			"InvalidNowInput",
			initialMiners:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			miner: 			"m2",
			now: 			0,
			expected:		false,
		},
		{
			name: 			"minerNotFoundInDynasty",
			initialMiners:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			miner: 			"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2cf",
			now: 			90,
			expected:		false,
		},
	}

	for _,tt := range tests {
		t.Run(tt.name, func(t *testing.T){
			dynasty:= NewDynastyWithMiners(tt.initialMiners)
			nextMintTime := dynasty.IsMyTurn(tt.miner, tt.now)
			assert.Equal(t, tt.expected, nextMintTime)
		})
	}
}