package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/time/rate"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

var tracecount atomic.Uint32
var spancount atomic.Uint32

func reportStats(done chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
			time.Sleep(5 * time.Second)
			tc := tracecount.Swap(0)
			sc := spancount.Swap(0)
			fmt.Printf("Finished %d traces/s, %d spans/second.\n", tc/5, sc/5)
		}
	}
}

// On ECS with TCP, the agent host can be found in the ECS_CONTAINER_METADATA_FILE
// https://docs.datadoghq.com/containers/amazon_ecs/apm/?tab=ec2metadataendpoint#code
func retrieveDDAgentHostECS() (string, error) {
	filePath := os.Getenv("ECS_CONTAINER_METADATA_FILE")
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	var metadata struct {
		HostPrivateIPv4Address string `json:"HostPrivateIPv4Address"`
	}
	if err := json.Unmarshal(fileContent, &metadata); err != nil {
		return "", fmt.Errorf("failed to unmarshal ECS metadata: %v, content: %v", err, string(fileContent))
	}
	if metadata.HostPrivateIPv4Address == "" {
		return "", fmt.Errorf("HostPrivateIPv4Address is empty, content: %v", string(fileContent))
	}
	return metadata.HostPrivateIPv4Address, nil
}

func main() {
	tps := flag.Float64("tps", 1, "Target number of traces to generate per second.")
	spt := flag.Uint64("spt", 2, "Number of spans to put in each trace. (>=1)")
	testDuration := flag.Duration("testtime", 0, "Amount of time to run the test. A value of '0' means the test will continue indefinitely.")
	flag.Parse()

	var err error
	if v, ok := os.LookupEnv("TRACEGEN_TPS"); ok {
		*tps, err = strconv.ParseFloat(v, 64)
		if err != nil {
			panic(fmt.Sprintf("Invalid TRACEGEN_TPS=%v: %v", v, err))
		}
	}
	if v, ok := os.LookupEnv("TRACEGEN_SPT"); ok {
		*spt, err = strconv.ParseUint(v, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("Invalid TRACEGEN_SPT=%v: %v", v, err))
		}
	}
	if v, ok := os.LookupEnv("TRACEGEN_TESTTIME"); ok {
		*testDuration, err = time.ParseDuration(v)
		if err != nil {
			panic(fmt.Sprintf("Invalid TRACEGEN_TESTTIME=%v: %v", v, err))
		}
	}

	var opts []tracer.StartOption
	if v, ok := os.LookupEnv("ECS_AGENT_HOST"); ok && v == "true" {
		host, err := retrieveDDAgentHostECS()
		if err != nil {
			panic(fmt.Sprintf("Failed to retrieve DD agent host: %v", err))
		}
		os.Setenv("DD_AGENT_HOST", host)
		opts = append(opts, tracer.WithAgentAddr(host))
	}

	tracer.Start()
	defer tracer.Stop()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		<-sigs
		close(done)
		fmt.Println("Exiting tracegen.")
		os.Exit(0)
	}()

	// Sleeping is expensive and inaccurate, so for high trace/s rates, we need to send bursts.
	// Instead of send, sleep, send, sleep, etc. we will calculate the number of traces to send
	// per 100ms, and send that many traces every 100ms using the rate.Limiter.
	// 100ms seems to be a good period allowing us to spread trace generation while also
	// remaining accurate up to very high trace/s rates

	// tperloop is the max traces we can send per loop iteration (every ~1/10th of a second - 100 ms).
	// The rate limiter helpfully adjusts the timing to give exactly tps, at tperloop
	// per iteration. ceil makes sure our period is always >= 100ms
	tperloop := int(math.Ceil(*tps / 10))
	lim := rate.NewLimiter(rate.Limit(*tps), tperloop)

	testStart := time.Now()
	go reportStats(done)
	fmt.Printf("Sending %v Traces/s, each with %d spans.\n", *tps, *spt)
	for {
		select {
		case <-done:
			return
		default:
			istart := time.Now()
			if *testDuration > 0 && istart.After(testStart.Add(*testDuration)) {
				return
			}
			lim.WaitN(context.Background(), tperloop)
			for sel := 0; sel < tperloop; sel++ {
				switch sel % 2 {
				case 0:
					genChain(*spt)
				case 1:
					genFlat(*spt)
				}
			}
		}
	}
}

// genChain generates a trace with spans count of spans in it.
// The trace is structured with each span N being the child of span N-1.
func genChain(spans uint64) {
	sp := tracer.StartSpan("tracegen_chain")
	for i := uint64(1); i < spans; i++ {
		defer sp.Finish()
		spancount.Add(1)
		sp = tracer.StartSpan(fmt.Sprintf("tracegen_chain(%d)", i),
			tracer.ChildOf(sp.Context()))
	}
	sp.Finish()
	spancount.Add(1)
	tracecount.Add(1)
}

// genFlat generates a trace with spans count of spans in it.
// The trace is structured with one root span, with all other spans being
// children of that root.
func genFlat(spans uint64) {
	const traceDuration = 1 * time.Second
	tdelta := traceDuration / time.Duration(spans) // Duration of each child span
	start := time.Now()
	root := tracer.StartSpan("tracegen_flat")
	defer func() {
		root.Finish(tracer.FinishTime(start.Add(traceDuration)))
		spancount.Add(1)
		tracecount.Add(1)
	}()
	for i := uint64(1); i < spans; i++ {
		sp := tracer.StartSpan(fmt.Sprintf("tracegen_flat(%d)", i),
			tracer.StartTime(start.Add(tdelta*time.Duration(i))),
			tracer.ChildOf(root.Context()))
		sp.Finish(tracer.FinishTime(start.Add(tdelta*time.Duration(i) + tdelta)))
		spancount.Add(1)
	}
}
