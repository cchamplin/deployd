package metrics

import (
	"sync"
	"time"
)

type Metrics struct {
	AverageTime       int64
	TotalMeasurements int64
	mutex             *sync.Mutex
}

type Metric struct {
	start int64
}

func NewMetrics() *Metrics {
	m := Metrics{}
	m.mutex = &sync.Mutex{}
	m.AverageTime = 0
	m.TotalMeasurements = 0
	return &m
}

func (m *Metrics) StartMeasure() Metric {
	now := time.Now()
	start := now.Unix()
	result := Metric{start: start}
	return result
}

func (m *Metrics) StopMeasure(metric Metric) {
	elapsed := time.Now().Unix() - metric.start
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.TotalMeasurements == 0 {
		m.AverageTime = elapsed
		m.TotalMeasurements = 1
	} else {
		newAvg := (m.AverageTime * m.TotalMeasurements) + elapsed/(m.TotalMeasurements+1)
		m.AverageTime = newAvg
		m.TotalMeasurements++
	}
}

func (m *Metrics) PercentOfTotal(total *Metrics) int64 {
	return total.AverageTime / m.AverageTime
}
