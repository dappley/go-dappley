package metrics

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
)

type stat struct {
	Timestamp int64       `json:"timestamp"`
	Value     interface{} `json:"value"`
}

type metric struct {
	Stats  *common.EvictingQueue `json:"stats"`
	update func() interface{}
}

type dataStore struct {
	Metrics      map[string]*metric `json:"metrics"`
	statCapacity int
	interval     time.Duration
	tasksStarted bool
	quit         chan bool
	mutex        *sync.RWMutex
}

// newDataStore constructor from dataStore
// statCapacity : maximum number of stats to store for any given metric
// interval : interval at which to collect new stats
func newDataStore(statCapacity int, interval time.Duration) *dataStore {
	return &dataStore{
		Metrics:      make(map[string]*metric),
		statCapacity: statCapacity,
		interval:     interval,
		tasksStarted: false,
		quit:         make(chan bool, 1),
		mutex:        &sync.RWMutex{},
	}
}

// String returns the json string representation of a dataStore to implement expvar.Var interface
func (ds *dataStore) String() string {
	ds.mutex.RLock()
	bytes, err := json.Marshal(ds)
	ds.mutex.RUnlock()
	if err != nil {
		logrus.Warn(err)
		return "null"
	}
	return string(bytes)
}

// registerNewMetric returns nil on success or an error if attempting to register a metric that already exists
// name: unique id of metric
// updateMetric: function that returns the value of the metric at any given time
func (ds *dataStore) registerNewMetric(name string, updateMetric func() interface{}) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	if _, ok := ds.Metrics[name]; ok {
		return errors.New("unable to register duplicate metric")
	}

	ds.Metrics[name] = &metric{common.NewEvictingQueue(ds.statCapacity), updateMetric}
	return nil
}

// startUpdate starts collection on registered metrics
func (ds *dataStore) startUpdate() {
	if !ds.tasksStarted {
		go func() {
			tick := time.NewTicker(ds.interval)
			defer tick.Stop()
			for {
				select {
				case t := <-tick.C:
					ds.mutex.Lock()
					for _, metric := range ds.Metrics {
						metric.Stats.Push(stat{t.Unix(), metric.update()})
					}
					ds.mutex.Unlock()
				case <-ds.quit:
					return
				}
			}
		}()

		ds.tasksStarted = true
	}
}

// stopUpdate stops collection on registered metrics
func (ds *dataStore) stopUpdate() {
	ds.quit <- true
	ds.tasksStarted = false
}

// getNumStats returns the number of collected stats for a given metric
func (ds *dataStore) getNumStats(metric string) int {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	return ds.Metrics[metric].Stats.Len()
}
