package main

import (
	"flag"
	"log"
	"math"
	"net/http"
	"time"
)

func main() {
	url := flag.String("url", "http://nginx", "URL")
	minTPS := flag.Float64("min-tps", 10.0, "minimum number of queries per second")
	maxTPS := flag.Float64("max-tps", 50.0, "maximum number of queries per second")
	period := flag.Duration("period", 30*time.Minute, "period of the sine wave of queries TPS")
	flag.Parse()

	amplitude := (*maxTPS - *minTPS) / 2.0

	tr := &http.Transport{
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	for {
		go func() {
			resp, err := client.Get(*url)
			if err != nil {
				log.Printf("http.Get(%s) -> %s", *url, err)
				return
			}
			defer resp.Body.Close()
		}()

		s := math.Sin(2 * math.Pi * float64(time.Now().Unix()) / period.Seconds())
		p := time.Duration(1000000000 / (*minTPS + amplitude + amplitude*s))
		time.Sleep(p)
	}
}
