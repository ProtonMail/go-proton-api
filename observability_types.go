package proton

type ObservabilityBatch struct {
	Metrics []ObservabilityMetric `json:"Metrics"`
}

type ObservabilityMetric struct {
	Name      string      `json:"Name"`
	Version   int         `json:"Version"`
	Timestamp int64       `json:"Timestamp"` // Unix timestamp
	Data      interface{} `json:"Data"`

	ShouldCache bool `json:"-"` // Internal field, indicating whether we should cache the given observability metric.
}
