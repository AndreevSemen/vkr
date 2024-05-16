package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"gossip_emulation/internal/spread/agent/broadcast"
	"gossip_emulation/internal/spread/agent/gossip"
	"gossip_emulation/internal/spread/cluster/local"
	"gossip_emulation/internal/spread/types"
)

func BenchmarkBroadcast(b *testing.B) {
	b.StopTimer()
	b.ResetTimer()

	var maxMsgsSum int
	var iters int
	for i := 0; i < 100; i++ {
		iters++
		maxMsgsSum += benchBroadcastCluster(b, b.N)
		runtime.GC()
	}

	b.ReportMetric(float64(maxMsgsSum)/float64(iters), "avg-max-msgs")
}

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
		// if n >= 500 {
		// 	return n + 10
		// }
		// if n >= 200 {
		// 	return n + 5
		// }
		if n >= 100 {
			return n + 2
		}

		return n + 1
	}

	records := [][]string{
		{"id", "n", "L"},
	}
	rounds := 20
	for i := 1; i <= 2500; i = incCounter(i) {
		if i == 0 {
			continue
		}
		m := -1
		for j := 0; j < rounds; j++ {
			maxMsgs := benchGossipCluster(b, i)
			if m == -1 || m > maxMsgs {
				m = maxMsgs
			}
			runtime.GC()
		}

		fmt.Printf("%d,%d,%d\n", i, i, m)
		records = append(records, []string{
			strconv.Itoa(i),
			strconv.Itoa(i),
			strconv.Itoa(m),
		})

		if i%100 == 0 {
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

func benchBroadcastCluster(b *testing.B, n int) (maxAgentsMsgs int) {
	cl, ags := initBroadcastCluster(b, n)
	defer func() {
		b.ReportMetric(float64(cl.SentMessages()), "msgs")
	}()
	return spreadCluster(b, cl, ags)
}

func benchGossipCluster(b *testing.B, n int) (maxAgentsMsgs int) {
	cl, ags := initGossipCluster(b, n, 3)
	defer func() {
		b.ReportMetric(float64(cl.SentMessages()), "msgs")
	}()
	return spreadCluster(b, cl, ags)
}

func initBroadcastCluster(b *testing.B, n int) (types.Cluster, []types.Agent) {
	failures := map[local.Edge]struct{}{}
	cl := local.NewCluster(failures)

	agentCfg := broadcast.BroadcastAgentConfig{
		SendTimeout: 200 * time.Millisecond,
		SeedAgents:  make([]string, 0, n),
	}

	for i := 0; i < n; i++ {
		agentCfg.SeedAgents = append(agentCfg.SeedAgents, strconv.Itoa(i))
	}

	agents := make([]types.Agent, 0, n)
	for i := 0; i < n; i++ {
		agentCfg.Address = strconv.Itoa(i)

		a, err := broadcast.NewBroadcastAgent(agentCfg, cl)
		require.NoError(b, err)

		agents = append(agents, a)
	}

	return cl, agents
}

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

	var heartbeat time.Duration = time.Duration(n*15/100) * time.Millisecond
	if n < 500 {
		heartbeat = 25 * time.Millisecond
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

func spreadCluster(b *testing.B, cl types.Cluster, ags []types.Agent) (maxAgentsMsgs int) {
	e1 := types.Event{
		UUID: uuid.New(),
	}

	wg := &sync.WaitGroup{}
	for i := range ags {
		a := ags[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for gotE := range a.GetEvents() {
				if gotE.UUID == e1.UUID {
					break
				}
			}
		}()
	}

	randAg1 := ags[rand.Intn(len(ags))]
	b.StartTimer()
	randAg1.PublishEvent(e1)
	wg.Wait()
	b.StopTimer()

	for _, a := range ags {
		require.NoError(b, a.Close(), "close agent")
	}

	return cl.MaxAgentMsgs()
}
