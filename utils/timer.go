package utils

import (
	"fmt"
	"strings"
	"time"
)

type Timer struct {
	name      string
	startTime time.Time
	endTime   time.Time
	duration  time.Duration
}

func NewTimer(name string) *Timer {
	return &Timer{
		name: name,
	}
}

func (t *Timer) Start() {
	t.startTime = time.Now()
}

func (t *Timer) Stop() float64 {
	t.endTime = time.Now()
	t.duration = t.endTime.Sub(t.startTime)
	return float64(t.duration.Nanoseconds())
}

func (t *Timer) GetDuration() float64 {
	return float64(t.duration.Nanoseconds())
}

func (t *Timer) String() string {
	return fmt.Sprintf("Timer{%s: %.0f ns}", t.name, t.GetDuration())
}

type PerformanceResult struct {
	Operation string
	Average   float64
}

func NewPerformanceResult(operation string) *PerformanceResult {
	return &PerformanceResult{
		Operation: operation,
	}
}

func (pr *PerformanceResult) AddTime(duration float64) {
	pr.Average = duration
}

func (pr *PerformanceResult) String() string {
	return fmt.Sprintf("Performance{%s: avg=%.0fns}",
		pr.Operation, pr.Average)
}

func (pr *PerformanceResult) PrintResults() {
	fmt.Printf("\n=== %s benchmark result ===\n", pr.Operation)
	fmt.Printf("Method: cumulative timing averaged over iterations\n")
	fmt.Printf("Average time: %.0f ns (%.3f ms)\n", pr.Average, pr.Average/1e6)
}

type PerformanceSuite struct {
	Results map[string]*PerformanceResult
}

func NewPerformanceSuite() *PerformanceSuite {
	return &PerformanceSuite{
		Results: make(map[string]*PerformanceResult),
	}
}

func (ps *PerformanceSuite) AddResult(operation string, duration float64) {
	if ps.Results[operation] == nil {
		ps.Results[operation] = NewPerformanceResult(operation)
	}
	ps.Results[operation].AddTime(duration)
}

func (ps *PerformanceSuite) PrintAllResults() {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("Benchmark summary\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	for _, result := range ps.Results {
		result.PrintResults()
	}

	fmt.Printf(strings.Repeat("=", 60) + "\n")
}
