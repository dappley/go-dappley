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
	compareJSON(t, `{"NumTxPerBlock":[1,2],"BlockHeights":[1,2]}`, bst)

	bst.Update(3, 3)
	compareJSON(t, `{"NumTxPerBlock":[2,3],"BlockHeights":[2,3]}`, bst)
}

func TestBlockStatsTracker_Filter(t *testing.T) {
	bst := NewBlockStatsTracker(1)
	bst.Update(1, 1)
	compareJSON(t, `{"NumTxPerBlock":[1],"BlockHeights":[1]}`, bst.Filter(true, true))
	compareJSON(t, `{"NumTxPerBlock":[0],"BlockHeights":[1]}`, bst.Filter(true, false))
}

func compareJSON(t *testing.T, expected string, obj interface{}) {
	bytes, err := json.Marshal(obj)
	assert.Nil(t, err)
	assert.JSONEq(t, expected, string(bytes))
}
