package consensus

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/core"
)

func TestDynasty_NewDynasty(t *testing.T) {
	dynasty := NewDynasty()
	assert.Empty(t,dynasty.producers)
}

func TestDynasty_NewDynastyWithProducers(t *testing.T) {
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
			dynasty:= NewDynastyWithProducers(tt.input)
			assert.Equal(t, tt.expected, dynasty.producers)
		})
	}
}

func TestDynasty_AddProducer(t *testing.T) {
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
			dynasty.AddProducer(tt.input)
			assert.Equal(t, tt.expected, dynasty.producers)
		})
	}
}

func TestDynasty_AddMultipleProducers(t *testing.T) {
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
			dynasty.AddMultipleProducers(tt.input)
			assert.Equal(t, tt.expected, dynasty.producers)
		})
	}
}

func TestDynasty_GetMinerIndex(t *testing.T) {
	tests := []struct{
		name             string
		initialProducers []string
		miner            string
		expected         int
	}{
		{
			name: 			"minerCouldBeFound",
			initialProducers:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			miner: 			"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
			expected:		0,
		},
		{
			name: 			"minerCouldNotBeFound",
			initialProducers:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			miner: 			"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDF",
			expected:		-1,
		},
		{
			name: 			"EmptyInput",
			initialProducers:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			miner: 			"",
			expected:		-1,
		},
	}

	for _,tt := range tests {
		t.Run(tt.name, func(t *testing.T){
			dynasty:= NewDynastyWithProducers(tt.initialProducers)
			index := dynasty.GetProducerIndex(tt.miner)
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
		name             string
		initialProducers []string
		producer         string
		index            int
		now              int64
		expected         bool
	}{
		{
			name: 			"IsMyTurn",
			initialProducers:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			producer: "1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct",
			now:      75,
			expected: true,
		},
		{
			name: 			"NotMyTurn",
			initialProducers:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			producer: "1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
			now:      61,
			expected: false,
		},
		{
			name: 			"EmptyInput",
			initialProducers:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			producer: "",
			now:      61,
			expected: false,
		},
		{
			name: 			"InvalidNowInput",
			initialProducers:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			producer: "m2",
			now:      0,
			expected: false,
		},
		{
			name: 			"minerNotFoundInDynasty",
			initialProducers:	[]string{
				"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
				"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"},
			producer: "1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2cf",
			now:      90,
			expected: false,
		},
	}

	for _,tt := range tests {
		t.Run(tt.name, func(t *testing.T){
			dynasty:= NewDynastyWithProducers(tt.initialProducers)
			nextMintTime := dynasty.IsMyTurn(tt.producer, tt.now)
			assert.Equal(t, tt.expected, nextMintTime)
		})
	}
}

func TestDynasty_ProducerAtATime(t *testing.T) {
	producers := []string{
		"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
		"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
		"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"}

	tests := []struct{
		name 		string
		now 		int64
		expected	string
	}{
		{
			name: 		"Normal",
			now: 		62,
			expected:	"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
		},
		{
			name: 		"InvalidInput",
			now: 		-1,
			expected:	"",
		},
	}

	for _,tt := range tests {
		t.Run(tt.name, func(t *testing.T){
			dynasty:= NewDynastyWithProducers(producers)
			producer := dynasty.ProducerAtATime(tt.now)
			assert.Equal(t, tt.expected, producer)
		})
	}
}

func TestDynasty_ValidateProducer(t *testing.T) {
	producers := []string{
		"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
		"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
		"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"}

	cbtx := core.NewCoinbaseTX("121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD","")
	cbtxInvalidProducer := core.NewCoinbaseTX("121yKAXeG4cw6uaGCBGjWk9yTWmMkhcoDD","")

	tests := []struct{
		name 		string
		block 		*core.Block
		expected	bool
	}{
		{
			name: 		"ValidProducer",
			block: 		core.FakeNewBlockWithTimestamp(
				46,
				[]*core.Transaction{
					core.MockTransaction(),
					&cbtx,
				},
				nil,
			),
			expected:	true,
		},
		{
			name: 		"ProducerNotAtItsTurn",
			block: 		core.FakeNewBlockWithTimestamp(
				44,
				[]*core.Transaction{
					core.MockTransaction(),
					&cbtx,
				},
				nil,
			),
			expected:	false,
		},
		{
			name: 		"NotAProducer",
			block: 		core.FakeNewBlockWithTimestamp(
				44,
				[]*core.Transaction{
					core.MockTransaction(),
					&cbtxInvalidProducer,
				},
				nil,
			),
			expected:	false,
		},
		{
			name: 		"EmptyBlock",
			block: 		nil,
			expected:	false,
		},
	}

	for _,tt := range tests {
		t.Run(tt.name, func(t *testing.T){
			dynasty:= NewDynastyWithProducers(producers)
			assert.Equal(t, tt.expected, dynasty.ValidateProducer(tt.block))
		})
	}
}
