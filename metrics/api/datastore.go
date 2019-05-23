package metrics

import (
    "time"
)

type stat struct {
    Timestamp int64 `json:"timestamp"`
    Value interface{} `json:"value"`
}

type metric struct {
    Stats []stat `json:"stats"`
    update func() interface{}
}

func (m *metric) setStats(stats []stat) {
    m.Stats = stats
}

type dataStore struct {
    Metrics map[string]*metric `json:"metrics"`
    statCapacity int
    interval time.Duration
    tasksStarted bool
}

func newDataStore(statCapacity int, interval time.Duration) *dataStore {
    return &dataStore{make(map[string]*metric), statCapacity, interval, false}
}

func (ds *dataStore) registerNewMetric(name string, updateMetric func() interface{}) {
    ds.Metrics[name] = &metric{[]stat{}, updateMetric}
}

func (ds *dataStore) startUpdate() {
    if !ds.tasksStarted {
        for k := range ds.Metrics {
            go func(key string) {
                for range time.NewTicker(ds.interval).C {
                    if len(ds.Metrics[key].Stats)+1 > ds.statCapacity {
                        ds.Metrics[key].setStats(ds.Metrics[key].Stats[1:])
                    }
                    ds.Metrics[key].setStats(append(ds.Metrics[key].Stats,
                        stat{time.Now().Unix(), ds.Metrics[key].update()}))
                }
            }(k)
        }

        ds.tasksStarted = true
    }
}
