package gb

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"os"
	"os/signal"
    "runtime/pprof"
)

func GetRandStr(length int) string {
    return genRandString(length)
}

func Prepare() {
    seed = uint32(time.Now().UnixNano())
}

type LatencyCounter [10]int

func (l *LatencyCounter) Add(nsec int64) {
    timeFloat := float64(nsec / 1e6 * 2)
    if timeFloat < 1 {
        l[0] += 1
        return
    }

	idx := int(math.Log10(timeFloat))
	if idx >= 10 {
		idx = 9
	}
	l[idx] += 1
}

func (l *LatencyCounter) merge(l2 *LatencyCounter) {
	for i := 0; i < 10; i++ {
		l[i] += l2[i]
	}
}

func (l *LatencyCounter) printLog() {
	fmt.Printf("\nLatency (ms)\n")
	for i := 0; i < 10; i++ {
		lagend := int(math.Pow(10, float64(i)/2.0))
		fmt.Printf("%d\t%d\n", lagend, l[i])
	}
}

type Bench struct {
	count uint

	numConcurrent, numTrials, numThrottles uint
	monitorLatency                        bool

	chDone chan int

	chCount        chan uint
	chThrottler    chan uint
	chLatency      chan *LatencyCounter
	latencyCounter *LatencyCounter

	chSignal chan os.Signal
    profilePath string
}

func New() (b *Bench) {
	b = new(Bench)
	b.count = 0

	b.chCount = make(chan uint, 100)
	b.chThrottler = make(chan uint, 100)
	b.chLatency = make(chan *LatencyCounter, 100)
	b.latencyCounter = new(LatencyCounter)

	b.chSignal = make(chan os.Signal, 1)
	signal.Notify(b.chSignal, os.Interrupt, os.Kill)

	b.init()

	return b
}

func (b *Bench) loop() {
	throttler := b.chThrottler
	_ = throttler

	prevCount := uint(0)
	startTime := time.Now().UnixNano()

	chTenth := time.Tick(time.Millisecond * 100)
	chSecond := time.Tick(time.Second)
	for {
		select {
		case latency := <-b.chLatency: // latency in nanoseconds
			b.latencyCounter.merge(latency)
		case count := <-b.chCount:
			b.count += count
		case <-chTenth:
			throttler = b.chThrottler
		case throttler <- b.numThrottles / 10:
			throttler = nil

		//reporter
		case <-chSecond:
			diff := b.count - prevCount
			fmt.Printf("%d\t%d\t%d\n", (time.Now().UnixNano()-startTime)/1e6, diff, b.count)
			prevCount = b.count

		//handle signal
		case <-b.chSignal:
            b.cleanup()
			os.Exit(-1)
		}
	}
}

func (b *Bench) run(runner BenchmarkRunner) {
	resolution := b.numThrottles / 10
	if resolution > 50 {
		resolution = 50
	}


	assigned := uint(0)
	count := uint(0)
	counter := new(LatencyCounter)

    defer func() {
        <-b.chDone // first, allow other goroutines to start

        // clean up statistics
        if count > 0 {
            b.chCount <- count
            if b.monitorLatency {
                b.chLatency <- counter
            }
        }

        // transfer quota to other goroutines
        b.chThrottler <- assigned
    }()

	for {
		assigned += <-b.chThrottler

		for assigned > 0 {
			assigned -= 1

			timeBefore := time.Now().UnixNano()

			done, err := runner.Execute()
			if err != nil {
				log.Fatalf("Failed to execute: %s\n", err.Error())
			}
			if done {
				return
			}

			if b.monitorLatency {
				counter.Add(time.Now().UnixNano() - timeBefore)
			}

			count += 1
			if count > resolution {
				b.chCount <- resolution
				count -= resolution

				if b.monitorLatency {
					b.chLatency <- counter
					counter = new(LatencyCounter)
				}
			}
		}
	}
}

func (b *Bench) init() {
	flag.UintVar(&b.numConcurrent, "concurrent", 50, "Number of concurrent executions")
	flag.UintVar(&b.numConcurrent, "c", 50, "Number of concurrent executions")
	flag.UintVar(&b.numTrials, "num", 1000, "Number of executions")
	flag.UintVar(&b.numTrials, "n", 1000, "Number of executions")
	flag.UintVar(&b.numThrottles, "throttle", 1000000, "Maximum number of queries per seconds")
	flag.UintVar(&b.numThrottles, "t", 1000000, "Maximum number of queries per seconds")

	flag.BoolVar(&b.monitorLatency, "b", true, "Monitor & print latency")
    flag.StringVar(&b.profilePath, "p", "", "Profiling path")

	flag.Parse()

	b.chDone = make(chan int, b.numConcurrent)
}

func (b *Bench) cleanup() {
	if b.monitorLatency {
		b.latencyCounter.printLog()
	}
    if b.profilePath != "" {
        pprof.StopCPUProfile()
    }
}

type BenchGenFunc func(seq uint) (r BenchmarkRunner, err error)

func (b *Bench) Run(gen BenchGenFunc) {
	rand.Seed(time.Now().UnixNano())
	go b.loop()

    if b.profilePath != "" {
        f, err := os.Create(b.profilePath)
        if err != nil {
            log.Fatal("Failed to create profile file: %s", err.Error())
        }
        pprof.StartCPUProfile(f)
    }

	for i := uint(0); i < b.numTrials; i++ {
		b.chDone <- 0
		runner, err := gen(i)
		if err != nil {
			panic(fmt.Sprintf("Failed to generate instance: %s", err.Error()))
		}

		go b.run(runner)
	}

	for i := uint(0); i < b.numConcurrent; i++ {
		b.chDone <- 0
    }

    b.cleanup()
}

type BenchmarkRunner interface {
	Execute() (done bool, err error)
}
