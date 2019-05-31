package metrics

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
)

type stat struct {
	Timestamp int64       `json:"timestamp"`
	Value     interface{} `json:"value"`
}

type metric struct {
	Stats  []stat `json:"stats"`
	update func() interface{}
}

func (m *metric) setStats(stats []stat) {
	m.Stats = stats
}

type dataStore struct {
	Metrics      map[string]*metric `json:"metrics"`
	statCapacity int
	interval     time.Duration
	tasksStarted bool
	quit         chan bool
}

// newDataStore constructor from dataStore
// statCapacity : maximum number of stats to store for any given metric
// interval : interval at which to collect new stats
func newDataStore(statCapacity int, interval time.Duration) *dataStore {
	return &dataStore{make(map[string]*metric), statCapacity, interval, false, make(chan bool, 1)}
}

// String returns the json string representation of a dataStore to implement expvar.Var interface
func (ds *dataStore) String() string {
	bytes, err := json.Marshal(ds)
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
	if _, ok := ds.Metrics[name]; ok {
		return errors.New("unable to register duplicate metric")
	}

	ds.Metrics[name] = &metric{make([]stat, 0, ds.statCapacity), updateMetric}
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
					for key := range ds.Metrics {
						if len(ds.Metrics[key].Stats)+1 > ds.statCapacity {
							ds.Metrics[key].setStats(ds.Metrics[key].Stats[1:])
						}
						ds.Metrics[key].setStats(append(ds.Metrics[key].Stats,
							stat{t.Unix(), ds.Metrics[key].update()}))
					}
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
