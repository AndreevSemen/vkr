package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
)

const (
	n = 1200
)

func main() {
	// f1, err := os.Open("bench.csv")
	// if err != nil {
	// 	panic(err)
	// }

	// r := csv.NewReader(f1)
	// raw, err := r.ReadAll()
	// if err != nil {}

	records := [][]string{
		{"id", "n", "L"},
	}
	for i := 0; i < n; i++ {
		records = append(records, []string{
			strconv.Itoa(i),
			strconv.Itoa(i),
			fmt.Sprintf("%.2f", 1*math.Log2(float64(i))),
		})
	}

	f, err := os.Create("log2(x).csv")
	if err != nil {
		panic(err)
	}
	w := csv.NewWriter(f)

	w.WriteAll(records)
	w.Flush()

	if err := w.Error(); err != nil {
		panic(err)
	}

	f.Close()

	records = [][]string{
		{"id", "n", "L"},
	}
	for i := 0; i < n; i++ {
		records = append(records, []string{
			strconv.Itoa(i),
			strconv.Itoa(i),
			fmt.Sprintf("%.2f", 1.7*math.Log2(float64(i))),
		})
	}

	f, err = os.Create("1.7log2(x).csv")
	if err != nil {
		panic(err)
	}
	w = csv.NewWriter(f)

	w.WriteAll(records)
	w.Flush()

	if err := w.Error(); err != nil {
		panic(err)
	}

	f.Close()
}
