package memoize

import "time"

type MetricEventKind uint8

const (
	MetricHit MetricEventKind = iota + 1
	MetricMiss
	MetricStaleHit
	MetricRefreshStart
	MetricRefreshSuccess
	MetricRefreshError
	MetricSet
	MetricDelete
)

type MetricEvent struct {
	Kind     MetricEventKind
	Key      string
	Duration time.Duration
	Err      error
}

type Metrics interface {
	RecordMetric(MetricEvent)
}

type noopMetrics struct{}

func (noopMetrics) RecordMetric(MetricEvent) {}
