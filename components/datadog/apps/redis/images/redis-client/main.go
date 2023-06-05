package main

import (
	"context"
	"flag"
	"log"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	addr := flag.String("addr", "redis:6379", "Redis server address")
	minTPS := flag.Float64("min-tps", 10.0, "minimum number of queries per second")
	maxTPS := flag.Float64("max-tps", 50.0, "maximum number of queries per second")
	period := flag.Duration("period", 30*time.Minute, "period of the sine wave of queries TPS")
	flag.Parse()

	amplitude := (*maxTPS - *minTPS) / 2.0

	for {
		go func() {
			rdb := redis.NewClient(&redis.Options{
				Addr:     *addr,
				Password: "", // no password set
				DB:       0,  // use default DB
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := rdb.Set(ctx, "key", "value", 0).Err()
			if err != nil {
				log.Printf("redis.Set(\"key\", \"value\") -> %s", err)
				return
			}

			val, err := rdb.Get(ctx, "key").Result()
			if err != nil {
				log.Printf("redis.Get(\"key\") -> %s, %s", val, err)
				return
			}
		}()

		s := math.Sin(2 * math.Pi * float64(time.Now().Unix()) / period.Seconds())
		p := 2 * time.Duration(1000000000/(*minTPS+amplitude+amplitude*s))
		time.Sleep(p)
	}
}
