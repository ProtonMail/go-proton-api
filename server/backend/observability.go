package backend

import (
	"time"

	"github.com/ProtonMail/go-proton-api"
)

type ObservabilityStatistics struct {
	Metrics     []proton.ObservabilityMetric
	RequestTime []time.Time
}

func NewObservabilityStatistics() ObservabilityStatistics {
	return ObservabilityStatistics{
		Metrics:     make([]proton.ObservabilityMetric, 0),
		RequestTime: make([]time.Time, 0),
	}
}

func (b *Backend) PushObservabilityMetrics(metrics []proton.ObservabilityMetric) {
	writeBackend(b, func(b *unsafeBackend) {
		b.observabilityStatistics.Metrics = append(b.observabilityStatistics.Metrics, metrics...)
		b.observabilityStatistics.RequestTime = append(b.observabilityStatistics.RequestTime, time.Now())
	})
}

func (b *Backend) GetObservabilityStatistics() ObservabilityStatistics {
	return readBackendRet(b, func(b *unsafeBackend) ObservabilityStatistics {
		return b.observabilityStatistics
	})
}
