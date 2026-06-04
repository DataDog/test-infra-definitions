package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/time/rate"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

var tracecount atomic.Uint32
var spancount atomic.Uint32
var additionalSpanTags map[string]string

func reportStats(done chan struct{}) {
	for {
		select {
		case <-done:
			return
		case <-time.After(5 * time.Second):
			tc := tracecount.Swap(0)
			sc := spancount.Swap(0)
			fmt.Printf("Finished %d traces/s, %d spans/second.\n", tc/5, sc/5)
		}
	}
}

// retrieveDDAgentHostECS retrieves the IP address of the ECS agent
// https://docs.datadoghq.com/containers/amazon_ecs/apm/?tab=ec2metadataendpoint#code
// We could use the ECS_CONTAINER_METADATA_FILE if the ECS Agent was configured to inject it.
func retrieveDDAgentHostECS() (string, error) {
	resp, err := http.Get("http://169.254.169.254/latest/meta-data/local-ipv4")
	if err != nil {
		return "", err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

func main() {
	tps := flag.Float64("tps", 1, "Target number of traces to generate per second.")
	spt := flag.Uint64("spt", 2, "Number of spans to put in each trace. (>=1)")
	testDuration := flag.Duration("testtime", 0, "Amount of time to run the test. A value of '0' means the test will continue indefinitely.")
	additionalSpanTagsFlat := flag.String("addspantags", "", "Comma separated list of additional tags to add to each span.")
	additionalSpanTags = make(map[string]string)
	waitDuration := flag.Duration("waittime", 0, "Amount of time to wait before the test actually starts producing data. Allows core-agent to detect new process and resolve container tags")
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
	if v, ok := os.LookupEnv("TRACEGEN_ADDSPANTAGS"); ok {
		*additionalSpanTagsFlat = v
	}
	if v, ok := os.LookupEnv("TRACEGEN_WAITTIME"); ok {
		*waitDuration, err = time.ParseDuration(v)
		if err != nil {
			panic(fmt.Sprintf("Invalid TRACEGEN_WAITTIME=%v: %v", v, err))
		}
	}

	if *additionalSpanTagsFlat != "" {
		for _, tagFull := range strings.Split(*additionalSpanTagsFlat, ",") {
			tagParts := strings.Split(tagFull, ":")
			if len(tagParts) < 1 || len(tagParts) > 2 {
				panic(fmt.Sprintf("Invalid addspantags flag=%v", tagFull))
			}

			tagKey := tagParts[0]
			tagVal := ""
			if len(tagParts) == 2 {
				tagVal = tagParts[1]
			}
			additionalSpanTags[tagKey] = tagVal
		}
		if len(additionalSpanTags) == 0 {
			panic(fmt.Sprintf("Invalid addspantags flag=%v", additionalSpanTagsFlat))
		}
	}

	var opts []tracer.StartOption
	if v, ok := os.LookupEnv("ECS_AGENT_HOST"); ok && v == "true" {
		host, err := retrieveDDAgentHostECS()
		if err != nil {
			panic(fmt.Sprintf("Failed to retrieve DD agent host: %v", err))
		}
		if host == "" {
			panic("Failed to retrieve DD agent host: no IP address found")
		}
		os.Setenv("DD_AGENT_HOST", host)
		opts = append(opts, tracer.WithAgentAddr(host))
	}

	tracer.Start()
	defer tracer.Stop()

	if *waitDuration > 0 {
		fmt.Printf("Sleeping %s before tracegen starts\n", waitDuration)
		time.Sleep(*waitDuration)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		<-sigs
		close(done)
		fmt.Println("Exiting tracegen.")
		time.Sleep(10 * time.Second) // alow time for other goroutines to shut down.
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
traceloop:
	for {
		select {
		case <-done:
			break traceloop
		default:
			istart := time.Now()
			if *testDuration > 0 && istart.After(testStart.Add(*testDuration)) {
				break traceloop
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
	sp := tracer.StartSpan("poison_pill", tracer.Tag(ext.ManualKeep, true))
	sp.Finish()
}

// genChain generates a trace with spans count of spans in it.
// The trace is structured with each span N being the child of span N-1.
func genChain(spans uint64) {
	sp := tracer.StartSpan("tracegen_chain")
	for tagKey, tagVal := range additionalSpanTags {
		sp.SetTag(tagKey, tagVal)
	}

	for i := uint64(1); i < spans; i++ {
		defer sp.Finish()
		spancount.Add(1)
		sp = tracer.StartSpan(fmt.Sprintf("tracegen_chain(%d)", i),
			tracer.ChildOf(sp.Context()))
		for tagKey, tagVal := range additionalSpanTags {
			sp.SetTag(tagKey, tagVal)
		}
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
	for tagKey, tagVal := range additionalSpanTags {
		root.SetTag(tagKey, tagVal)
	}
	defer func() {
		root.Finish(tracer.FinishTime(start.Add(traceDuration)))
		spancount.Add(1)
		tracecount.Add(1)
	}()
	for i := uint64(1); i < spans; i++ {
		sp := tracer.StartSpan(fmt.Sprintf("tracegen_flat(%d)", i),
			tracer.StartTime(start.Add(tdelta*time.Duration(i))),
			tracer.ChildOf(root.Context()))
		for tagKey, tagVal := range additionalSpanTags {
			sp.SetTag(tagKey, tagVal)
		}
		sp.Finish(tracer.FinishTime(start.Add(tdelta*time.Duration(i) + tdelta)))
		spancount.Add(1)
	}
}
