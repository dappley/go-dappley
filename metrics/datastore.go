package metrics

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
	metricspb "github.com/dappley/go-dappley/metrics/pb"
)

type stat struct {
	Timestamp int64               `json:"timestamp"`
	Value     metricspb.StatValue `json:"value"`
}

func (s stat) ToProto() *metricspb.Stat {
	return &metricspb.Stat{Timestamp: s.Timestamp, Value: s.Value}
}

type metric struct {
	Stats  *common.EvictingQueue `json:"stats"`
	update func() metricspb.StatValue
}

func (m metric) ToProto() *metricspb.Metric {
	stats := make([]*metricspb.Stat, 0, m.Stats.Len())
	m.Stats.ForEach(func(element common.Element) {
		stats = append(stats, element.(stat).ToProto())
	})
	return &metricspb.Metric{Stats: stats}
}

type DataStore struct {
	Metrics      map[string]*metric `json:"metrics"`
	statCapacity int
	interval     time.Duration
	tasksStarted bool
	quit         chan bool
	mutex        *sync.RWMutex
}

// newDataStore constructor from DataStore
// statCapacity : maximum number of stats to store for any given metric
// interval : interval at which to collect new stats
func NewDataStore(statCapacity int, interval time.Duration) *DataStore {
	return &DataStore{
		Metrics:      make(map[string]*metric),
		statCapacity: statCapacity,
		interval:     interval,
		tasksStarted: false,
		quit:         make(chan bool, 1),
		mutex:        &sync.RWMutex{},
	}
}

// String returns the json string representation of a DataStore to implement expvar.Var interface
func (ds *DataStore) String() string {
	ds.mutex.RLock()
	bytes, err := json.Marshal(ds)
	ds.mutex.RUnlock()
	if err != nil {
		logrus.Warn(err)
		return "null"
	}
	return string(bytes)
}

// RegisterNewMetric returns nil on success or an error if attempting to register a metric that already exists
// name: unique id of metric
// updateMetric: function that returns the value of the metric at any given time
func (ds *DataStore) RegisterNewMetric(name string, updateMetric func() metricspb.StatValue) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	if _, ok := ds.Metrics[name]; ok {
		return errors.New("unable to register duplicate metric")
	}

	ds.Metrics[name] = &metric{common.NewEvictingQueue(ds.statCapacity), updateMetric}
	return nil
}

// StartUpdate starts collection on registered metrics
func (ds *DataStore) StartUpdate() {
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

// StopUpdate stops collection on registered metrics
func (ds *DataStore) StopUpdate() {
	ds.quit <- true
	ds.tasksStarted = false
}

// getNumStats returns the number of collected stats for a given metric
func (ds *DataStore) getNumStats(metric string) int {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	return ds.Metrics[metric].Stats.Len()
}

func (ds *DataStore) ToProto() *metricspb.DataStore {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	metrics := make(map[string]*metricspb.Metric)
	for k, v := range ds.Metrics {
		metrics[k] = v.ToProto()
	}
	return &metricspb.DataStore{Metrics: metrics}
}
