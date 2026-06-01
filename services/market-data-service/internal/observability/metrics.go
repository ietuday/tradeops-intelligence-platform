package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry           *prometheus.Registry
	TicksReceived      prometheus.Counter
	TicksValid         prometheus.Counter
	TicksInvalid       prometheus.Counter
	TicksPublished     prometheus.Counter
	ProcessingDuration prometheus.Histogram
	MQTTStatus         prometheus.Gauge
	KafkaPublishErrors prometheus.Counter
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	metrics := &Metrics{
		Registry: registry,
		TicksReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "market_ticks_received_total",
			Help: "Total number of raw market ticks received from MQTT.",
		}),
		TicksValid: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "market_ticks_valid_total",
			Help: "Total number of valid market ticks processed.",
		}),
		TicksInvalid: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "market_ticks_invalid_total",
			Help: "Total number of invalid market ticks rejected.",
		}),
		TicksPublished: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "market_ticks_published_total",
			Help: "Total number of normalized market ticks published to Kafka.",
		}),
		ProcessingDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "market_tick_processing_duration_seconds",
			Help:    "Market tick processing duration in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
		}),
		MQTTStatus: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mqtt_connection_status",
			Help: "MQTT connection status. 1 means connected, 0 means disconnected.",
		}),
		KafkaPublishErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "kafka_publish_errors_total",
			Help: "Total number of Kafka publish errors.",
		}),
	}
	registry.MustRegister(metrics.TicksReceived, metrics.TicksValid, metrics.TicksInvalid, metrics.TicksPublished, metrics.ProcessingDuration, metrics.MQTTStatus, metrics.KafkaPublishErrors)
	return metrics
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

func (m *Metrics) ObserveProcessing(start time.Time) {
	m.ProcessingDuration.Observe(time.Since(start).Seconds())
}
