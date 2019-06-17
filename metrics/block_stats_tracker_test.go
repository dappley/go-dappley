package dapmetrics

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlockStatsTracker_Update(t *testing.T) {
	bst := NewBlockStatsTracker(2)
	bst.Update(1, 1)
	bst.Update(2, 2)
	expectedJSON :=
		`
	[
		{	
			"NumTransactions": 1,
			"Height": 1
		},
		{
			"NumTransactions": 2,
			"Height": 2
		}
	]
	`
	compareJSON(t, expectedJSON, bst)

	bst.Update(3, 3)
	expectedJSON =
		`
	[
		{	
			"NumTransactions": 2,
			"Height": 2
		},
		{
			"NumTransactions": 3,
			"Height": 3
		}
	]
	`
	compareJSON(t, expectedJSON, bst)
}

func compareJSON(t *testing.T, expected string, obj interface{}) {
	bytes, err := json.Marshal(obj)
	assert.Nil(t, err)
	assert.JSONEq(t, expected, string(bytes))
}
