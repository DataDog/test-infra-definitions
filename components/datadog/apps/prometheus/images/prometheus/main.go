package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	port := flag.Uint64("port", 8080, "TCP port number of the OpenMetrics server")
	period := flag.Duration("period", 5*time.Minute, "Period of the sine wave data")
	nbSeries := flag.Uint64("nb-series", 10, "Number of time series to emit")
	flag.Parse()

	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "prom_counter",
		Help: "Prometheus Counter",
	})

	gauges := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "prom_gauge",
		Help: "Prometheus Gauge",
	},
		[]string{"series"},
	)

	registry := prometheus.NewRegistry()

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		counter,
		gauges,
	)

	go func() {
		for {
			counter.Inc()
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		for {
			for i := uint64(0); i < *nbSeries; i++ {
				gauges.WithLabelValues(strconv.FormatUint(i, 10)).Set(math.Sin(2 * math.Pi * (float64(time.Now().Unix())/period.Seconds() + float64(i)/float64(*nbSeries))))
			}
			time.Sleep(1 * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(
		registry,
		promhttp.HandlerOpts{
			Registry:          registry,
			EnableOpenMetrics: true,
		}),
	)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
