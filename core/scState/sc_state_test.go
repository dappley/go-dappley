package scState

import (
	"errors"
	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core/stateLog"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScState_Save(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := utxo.NewUTXOCache(db)

	testState := []struct {
		name     string
		address  string
		key      string
		value    string
		block    hash.Hash
		expected interface{}
		statelog map[string]map[string]string
	}{
		{
			name:     "Input Data1",
			address:  "dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf",
			key:      "Account1",
			value:    "99",
			block:    util.Str2bytes("blkHash1"),
			expected: "99",
			statelog: map[string]map[string]string{
				"dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf": {"Account1": ""},
			},
		},
		{
			name:     "Input Data2",
			address:  "dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf",
			key:      "Account1",
			value:    "199",
			block:    util.Str2bytes("blkHash2"),
			expected: "199",
			statelog: map[string]map[string]string{
				"dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf": {"Account1": "99"},
			},
		},
		{
			name:     "Input Data3",
			address:  "dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf",
			key:      "Account2",
			value:    "299",
			block:    util.Str2bytes("blkHash3"),
			expected: "299",
			statelog: map[string]map[string]string{
				"dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf": {"Account2": ""},
			},
		},
		{
			name:     "Input Data4",
			address:  "dUuPPYshbBgkzUrgScEHWvdGbSxC8z4R12",
			key:      "Account3",
			value:    "399",
			block:    util.Str2bytes("blkHash4"),
			expected: "399",
			statelog: map[string]map[string]string{
				"dUuPPYshbBgkzUrgScEHWvdGbSxC8z4R12": {"Account3": ""},
			},
		},
		{
			name:     "Delet Data",
			address:  "dUuPPYshbBgkzUrgScEHWvdGbSxC8z4R12",
			key:      "Account3",
			value:    ScStateValueIsNotExist,
			block:    util.Str2bytes("blkHash5"),
			expected: errors.New("key is invalid"),
			statelog: map[string]map[string]string{
				"dUuPPYshbBgkzUrgScEHWvdGbSxC8z4R12": {"Account3": "399"},
			},
		},
	}

	for _, tt := range testState {
		t.Run(tt.name, func(t *testing.T) {
			scState := NewScState(cache)
			scState.SetStateValue(tt.address, tt.key, tt.value)
			assert.Nil(t, scState.Save(tt.block))

			valBytes, err := db.Get(util.Str2bytes(utxo.ScStateMapKey + tt.address + tt.key))
			if err == nil {
				assert.Equal(t, tt.expected, util.Bytes2str(valBytes))
			} else {
				assert.Equal(t, tt.expected, err)
			}

			stLogBytes, err := db.Get(util.Str2bytes(utxo.ScStateLogKey + util.Bytes2str(tt.block)))
			assert.Nil(t, err)
			assert.Nil(t, err)
			assert.Equal(t, tt.statelog, stateLog.DeserializeStateLog(stLogBytes).Log)

		})
	}

	valBytes, err := db.Get(util.Str2bytes(utxo.ScStateMapKey + testState[0].address + testState[0].key))
	assert.Nil(t, err)
	assert.Equal(t, testState[1].value, util.Bytes2str(valBytes))

	valBytes, err = db.Get(util.Str2bytes(utxo.ScStateMapKey + testState[2].address + testState[2].key))
	assert.Nil(t, err)
	assert.Equal(t, testState[2].value, util.Bytes2str(valBytes))

}

func TestScState_getStateLog(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := utxo.NewUTXOCache(db)

	stLog := stateLog.NewStateLog()
	stLog.Log = map[string]map[string]string{"dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf": {"Account1": "99"}}
	assert.Nil(t, db.Put(util.Str2bytes(utxo.ScStateLogKey+"blkHash"), stLog.SerializeStateLog()))

	scState := NewScState(cache)
	assert.Equal(t, stLog, scState.getStateLog(util.Str2bytes("blkHash")))
}
