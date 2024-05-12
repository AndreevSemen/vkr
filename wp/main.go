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
		{"id", "n", "log2(n)", "log3(n)", "log4(n)"},
	}
	for i := 1; i < n; i++ {
		records = append(records, []string{
			strconv.Itoa(i),
			strconv.Itoa(i),
			fmt.Sprintf("%.2f", math.Log2(float64(i))/math.Log2(2)),
			fmt.Sprintf("%.2f", math.Log2(float64(i))/math.Log2(3)),
			fmt.Sprintf("%.2f", math.Log2(float64(i))/math.Log2(4)),
		})
	}

	f, err := os.Create("log-bases-merged.csv")
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
}
