package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

func main() {
	sleep := flag.Duration("sleep", 1*time.Second, "Sleep duration between each dogstatsd data point")
	period := flag.Duration("period", 5*time.Minute, "Period of the sine wave data")
	nbSeries := flag.Uint("nb-series", 10, "Number of time series to emit")
	flag.Parse()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1, syscall.SIGUSR2)

	statsdClient, err := statsd.New(os.Getenv("STATSD_URL"))
	if err != nil {
		log.Fatal(err)
	}

	for {
		for i := uint(0); i < *nbSeries; i++ {
			statsdClient.Gauge("custom.metric",
				math.Sin(2*math.Pi*(float64(time.Now().Unix())/period.Seconds()+float64(i)/float64(*nbSeries))),
				[]string{fmt.Sprintf("series:%d", i)},
				1)
		}

		select {
		case s := <-c:
			opt := []statsd.Option{}
			switch s {
			case syscall.SIGUSR1:
				log.Println("Switching to a dogstatsd client that doesn’t aggregate")

				opt = []statsd.Option{
					statsd.WithoutClientSideAggregation(),
				}

			case syscall.SIGUSR2:
				log.Println("Switching to a dogstatsd client that aggregates")
			}
			statsdClient, err = statsd.New(os.Getenv("STATSD_URL"), opt...)
			if err != nil {
				log.Fatal(err)
			}
		default:
		}

		time.Sleep(*sleep)
	}
}
