package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDataStore_String(t *testing.T) {
	t.Parallel()
	ds := newDataStore(1, time.Second)
	assert.Equal(t, "{\"metrics\":{}}", ds.String())

	// register new metric
	err := ds.registerNewMetric("test", func() interface{} { return 1 })
	assert.Nil(t, err)
	assert.Equal(t, "{\"metrics\":{\"test\":{\"stats\":[]}}}", ds.String())
}

func TestDataStore_StringError(t *testing.T) {
	t.Parallel()
	ds := newDataStore(1, time.Second)
	err := ds.registerNewMetric("new.test", func() interface{} { return func() {} })
	assert.Nil(t, err)
	ds.startUpdate()

	// allow time for statistic generation
	time.Sleep(3 * time.Second)

	// new.test's update returns a function which returns an unsupported type error from json.Marshal
	assert.Equal(t, "null", ds.String())
}

func TestDataStoreCapacityConstraint(t *testing.T) {
	t.Parallel()
	ds := newDataStore(1, time.Second)
	err := ds.registerNewMetric("test", func() interface{} { return 1 })
	assert.Nil(t, err)

	ds.startUpdate()
	time.Sleep(3 * time.Second)
	// ensure capacity constraint is not violated
	assert.Equal(t, 1, len(ds.Metrics["test"].Stats))
}

func TestDataStore_RegisterNewMetric(t *testing.T) {
	t.Parallel()
	ds := newDataStore(1, time.Second)

	err := ds.registerNewMetric("test", func() interface{} { return 0 })
	assert.Nil(t, err)

	err = ds.registerNewMetric("test", func() interface{} { return 1 })
	assert.Equal(t, "unable to register duplicate metric", err.Error())
}

func TestDataStore_Update(t *testing.T) {
	t.Parallel()
	ds := newDataStore(5, time.Second)

	err := ds.registerNewMetric("test", func() interface{} { return 1 })
	assert.Nil(t, err)

	ds.startUpdate()
	time.Sleep(2 * time.Second)
	ds.stopUpdate()

	// in case we need to wait for last collected stat
	time.Sleep(time.Second)

	// ensure some stats were collected
	numStats := len(ds.Metrics["test"].Stats)
	assert.True(t, numStats > 0)

	// test stop update
	time.Sleep(2 * time.Second)
	assert.Equal(t, numStats, len(ds.Metrics["test"].Stats))

	// test restart
	ds.startUpdate()
	time.Sleep(2 * time.Second)
	assert.True(t, len(ds.Metrics["test"].Stats) > numStats)
	ds.stopUpdate()
	time.Sleep(time.Second)
	numStats = len(ds.Metrics["test"].Stats)
	time.Sleep(2 * time.Second)
	assert.Equal(t, numStats, len(ds.Metrics["test"].Stats))
}
