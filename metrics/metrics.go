package metrics

import (
	"net/http"

	"github.com/Allenxuxu/toolkit/sync/atomic"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const defaultMetricsPath = "/metrics"

var (
	Enable atomic.Bool
	rg     = prometheus.NewRegistry()
)

var (
	bufferType = "type"
	perAddr    = "per_addr"

	ConnBufferLen = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gev_conn_buffer_len",
	}, []string{bufferType, perAddr})
	ConnBufferCap = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gev_conn_buffer_cap",
	}, []string{bufferType, perAddr})

	ConnHandlerDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gev_conn_duration_microseconds",
		},
		[]string{"action"},
	)

	DoPendingFuncDuration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gev_pending_func_duration_microseconds",
		},
	)
)

func PrometheusMustRegister(cs ...prometheus.Collector) {
	rg.MustRegister(cs...)
}

func MustRun(path, address string) {
	if path == "" {
		path = defaultMetricsPath
	}

	rg.MustRegister(
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		prometheus.NewGoCollector(),
		ConnBufferLen,
		ConnBufferCap,
		ConnHandlerDuration,
		DoPendingFuncDuration,
	)

	Enable.Set(true)
	defer Enable.Set(false)

	http.Handle(path, promhttp.HandlerFor(rg, promhttp.HandlerOpts{}))
	if err := http.ListenAndServe(address, nil); err != nil {
		panic(err)
	}
}
