package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"gossip_emulation/spread/agent/gossip"
	"gossip_emulation/spread/cluster/local"
	"gossip_emulation/spread/types"
)

// func BenchmarkBroadcast(b *testing.B) {
// 	b.StopTimer()
// 	b.ResetTimer()

// 	var maxMsgsSum int
// 	var iters int
// 	for i := 0; i < 100; i++ {
// 		iters++
// 		maxMsgsSum += benchBroadcastCluster(b, b.N)
// 		runtime.GC()
// 	}

// 	b.ReportMetric(float64(maxMsgsSum)/float64(iters), "avg-max-msgs")
// }

func BenchmarkGossip(b *testing.B) {
	b.StopTimer()
	b.ResetTimer()

	// var maxMsgsSum int
	// var iters int
	// for i := 0; i < 100; i++ {
	// 	iters++
	// 	maxMsgsSum += benchGossipCluster(b, b.N)
	// 	runtime.GC()
	// }

	incCounter := func(n int) int {
		if n >= 100 {
			return n + 10
		}

		return n + 10
	}

	records := [][]string{
		{"id", "n", "p100", "p95", "p90", "p85", "p80", "p75", "p70", "p65", "p60", "p55", "p50", "p45", "p40", "p35", "p30", "p25", "p20", "p15", "p10", "p05"},
	}
	rounds := 10
	for i := 0; i <= 2500; i = incCounter(i) {
		if i == 0 {
			continue
		}

		times := [20]time.Duration{}
		for j := 0; j < rounds; j++ {
			currTimes := benchGossipCluster(b, i)
			for i := range currTimes {
				times[i] += currTimes[i]
			}
			runtime.GC()
		}

		record := []string{
			strconv.Itoa(i),
			strconv.Itoa(i),
		}
		for i := range times {
			record = append(record,
				fmt.Sprintf("%.3f", (times[i]/time.Duration(rounds)).Seconds()),
			)
		}

		fmt.Println(strings.Join(record, ","))
		records = append(records, record)

		if i%25 == 0 {
			f, err := os.Create("bench.csv")
			if err != nil {
				b.Fatalf("create file: %s", err.Error())
			}
			w := csv.NewWriter(f)

			w.WriteAll(records)
			w.Flush()

			if err := w.Error(); err != nil {
				b.Fatalf("write to file: %s", err.Error())
			}

			f.Close()
		}
	}

	// b.ReportMetric(float64(maxMsgsSum)/float64(iters), "avg-max-msgs")
}

// func benchBroadcastCluster(b *testing.B, n int) [11]time.Duration {
// 	cl, ags := initBroadcastCluster(b, n)
// 	defer func() {
// 		b.ReportMetric(float64(cl.SentMessages()), "msgs")
// 	}()
// 	return spreadCluster(b, cl, ags)
// }

func benchGossipCluster(b *testing.B, n int) [20]time.Duration {
	cl, ags := initGossipCluster(b, n, 3)
	defer func() {
		b.ReportMetric(float64(cl.SentMessages()), "msgs")
	}()
	return spreadCluster(b, cl, ags)
}

// func initBroadcastCluster(b *testing.B, n int) (types.Cluster, []types.Agent) {
// 	failures := map[local.Edge]struct{}{}
// 	cl := local.NewCluster(failures)

// 	agentCfg := broadcast.BroadcastAgentConfig{
// 		SendTimeout: 200 * time.Millisecond,
// 		SeedAgents:  make([]string, 0, n),
// 	}

// 	for i := 0; i < n; i++ {
// 		agentCfg.SeedAgents = append(agentCfg.SeedAgents, strconv.Itoa(i))
// 	}

// 	agents := make([]types.Agent, 0, n)
// 	for i := 0; i < n; i++ {
// 		agentCfg.Address = strconv.Itoa(i)

// 		a, err := broadcast.NewBroadcastAgent(agentCfg, cl)
// 		require.NoError(b, err)

// 		agents = append(agents, a)
// 	}

// 	return cl, agents
// }

func initGossipCluster(b *testing.B, n int, fanout int) (types.Cluster, []types.Agent) {
	failures := map[local.Edge]struct{}{}
	// for i := 0; i < n; i++ {
	// 	edge := cluster.Edge{
	// 		From: strconv.Itoa(rand.Intn(n)),
	// 		To:   strconv.Itoa(rand.Intn(n)),
	// 	}
	// 	failures[edge] = struct{}{}
	// }

	cl := local.NewCluster(failures)

	var heartbeat time.Duration = time.Duration(n*8/100) * time.Millisecond
	if n < 5000 {
		heartbeat = 50 * time.Millisecond
	}
	// if n > 2000 {
	// 	heartbeat = 45 * time.Millisecond
	// } else if n > 1500 {
	// 	heartbeat = 35 * time.Millisecond
	// } else if n > 1000 {
	// 	heartbeat = 25 * time.Millisecond
	// } else if n > 500 {
	// 	heartbeat = 15 * time.Millisecond
	// } else {
	// 	heartbeat = 10 * time.Millisecond
	// }

	agentCfg := gossip.GossipAgentConfig{
		SendTimeout: 200 * time.Millisecond,
		SeedAgents:  make([]string, 0, n),
		Fanout:      fanout,
		Heartbeat:   heartbeat,
	}

	for i := 0; i < n; i++ {
		agentCfg.SeedAgents = append(agentCfg.SeedAgents, strconv.Itoa(i))
	}

	agents := make([]types.Agent, 0, n)
	for i := 0; i < n; i++ {
		agentCfg.Address = strconv.Itoa(i)

		a, err := gossip.NewGossipAgent(agentCfg, cl)
		require.NoError(b, err)

		agents = append(agents, a)
	}

	return cl, agents
}

func spreadCluster(b *testing.B, _ types.Cluster, ags []types.Agent) [20]time.Duration {
	e1 := types.Event{
		UUID: uuid.New(),
	}

	p100 := int64(len(ags))
	p95 := int64(math.Ceil(float64(len(ags)) * 0.95))
	p90 := int64(math.Ceil(float64(len(ags)) * 0.90))
	p85 := int64(math.Ceil(float64(len(ags)) * 0.85))
	p80 := int64(math.Ceil(float64(len(ags)) * 0.80))
	p75 := int64(math.Ceil(float64(len(ags)) * 0.75))
	p70 := int64(math.Ceil(float64(len(ags)) * 0.70))
	p65 := int64(math.Ceil(float64(len(ags)) * 0.65))
	p60 := int64(math.Ceil(float64(len(ags)) * 0.60))
	p55 := int64(math.Ceil(float64(len(ags)) * 0.55))
	p50 := int64(math.Ceil(float64(len(ags)) * 0.50))
	p45 := int64(math.Ceil(float64(len(ags)) * 0.45))
	p40 := int64(math.Ceil(float64(len(ags)) * 0.40))
	p35 := int64(math.Ceil(float64(len(ags)) * 0.35))
	p30 := int64(math.Ceil(float64(len(ags)) * 0.30))
	p25 := int64(math.Ceil(float64(len(ags)) * 0.25))
	p20 := int64(math.Ceil(float64(len(ags)) * 0.20))
	p15 := int64(math.Ceil(float64(len(ags)) * 0.15))
	p10 := int64(math.Ceil(float64(len(ags)) * 0.10))
	p05 := int64(math.Ceil(float64(len(ags)) * 0.05))

	var (
		time100 time.Time
		time95  time.Time
		time90  time.Time
		time85  time.Time
		time80  time.Time
		time75  time.Time
		time70  time.Time
		time65  time.Time
		time60  time.Time
		time55  time.Time
		time50  time.Time
		time45  time.Time
		time40  time.Time
		time35  time.Time
		time30  time.Time
		time25  time.Time
		time20  time.Time
		time15  time.Time
		time10  time.Time
		time05  time.Time
	)

	var mx sync.Mutex
	var completed int64
	wg := &sync.WaitGroup{}
	for i := range ags {
		a := ags[i]
		wg.Add(1)
		go func() {
			for gotE := range a.GetEvents() {
				if gotE.UUID == e1.UUID {
					break
				}
			}

			mx.Lock()
			now := time.Now()
			completed++
			completedNumber := completed
			mx.Unlock()

			if completedNumber == p100 {
				time100 = now
			}

			if completedNumber == p95 {
				time95 = now
			}

			if completedNumber == p90 {
				time90 = now
			}

			if completedNumber == p85 {
				time85 = now
			}

			if completedNumber == p80 {
				time80 = now
			}

			if completedNumber == p75 {
				time75 = now
			}

			if completedNumber == p70 {
				time70 = now
			}

			if completedNumber == p65 {
				time65 = now
			}

			if completedNumber == p60 {
				time60 = now
			}

			if completedNumber == p55 {
				time55 = now
			}

			if completedNumber == p50 {
				time50 = now
			}

			if completedNumber == p45 {
				time45 = now
			}

			if completedNumber == p40 {
				time40 = now
			}

			if completedNumber == p35 {
				time35 = now
			}

			if completedNumber == p30 {
				time30 = now
			}

			if completedNumber == p25 {
				time25 = now
			}

			if completedNumber == p20 {
				time20 = now
			}

			if completedNumber == p15 {
				time15 = now
			}

			if completedNumber == p10 {
				time10 = now
			}

			if completedNumber == p05 {
				time05 = now
			}

			wg.Done()
		}()
	}

	randAg1 := ags[rand.Intn(len(ags))]
	b.StartTimer()
	start := time.Now()
	randAg1.PublishEvent(e1)
	wg.Wait()
	b.StopTimer()

	for _, a := range ags {
		require.NoError(b, a.Close(), "close agent")
	}

	durations := [20]time.Duration{
		time100.Sub(start),
		time95.Sub(start),
		time90.Sub(start),
		time85.Sub(start),
		time80.Sub(start),
		time75.Sub(start),
		time70.Sub(start),
		time65.Sub(start),
		time60.Sub(start),
		time55.Sub(start),
		time50.Sub(start),
		time45.Sub(start),
		time40.Sub(start),
		time35.Sub(start),
		time30.Sub(start),
		time25.Sub(start),
		time20.Sub(start),
		time15.Sub(start),
		time10.Sub(start),
		time05.Sub(start),
	}

	return durations
}
