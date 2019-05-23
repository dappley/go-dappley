package metrics

import (
    "time"
)

type Stat struct {
    Timestamp int64 `json:"timestamp"`
    Value interface{} `json:"value"`
}

type Metric struct {
    Stats []Stat `json:"stats"`
    update func() interface{}
}

func (m *Metric) setStats(stats []Stat) {
    m.Stats = stats
}

type DataStore struct {
    Metrics map[string]*Metric `json:"metrics"`
    statCapacity int
    interval time.Duration
    tasksStarted bool
}

func NewDataStore(statCapacity int, interval time.Duration) *DataStore {
    return &DataStore{make(map[string]*Metric), statCapacity, interval, false}
}

func (ds *DataStore) RegisterNewMetric(name string, updateMetric func() interface{}) {
    ds.Metrics[name] = &Metric{[]Stat{}, updateMetric}
}

func (ds *DataStore) StartUpdate() {
    if !ds.tasksStarted {
        for k := range ds.Metrics {
            go func(key string) {
                for range time.NewTicker(ds.interval).C {
                    if len(ds.Metrics[key].Stats)+1 > ds.statCapacity {
                        ds.Metrics[key].setStats(ds.Metrics[key].Stats[1:])
                    }
                    ds.Metrics[key].setStats(append(ds.Metrics[key].Stats,
                        Stat{time.Now().Unix(), ds.Metrics[key].update()}))
                }
            }(k)
        }

        ds.tasksStarted = true
    }
}
