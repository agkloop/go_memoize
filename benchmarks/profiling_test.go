package benchmarks

import (
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
)

// TestMain enables opt-in CPU and memory profiling when BENCH_PROFILE=1.
// Usage: BENCH_PROFILE=1 go test ./benchmarks/ -bench=. -benchtime=10s
func TestMain(m *testing.M) {
	if os.Getenv("BENCH_PROFILE") != "1" {
		os.Exit(m.Run())
	}

	// CPU profile
	cpuF, err := os.Create("cpu.prof")
	if err != nil {
		panic(err)
	}
	if err := pprof.StartCPUProfile(cpuF); err != nil {
		panic(err)
	}

	code := m.Run()

	pprof.StopCPUProfile()
	_ = cpuF.Close()

	// Heap profile
	memF, err := os.Create("mem.prof")
	if err != nil {
		panic(err)
	}
	runtime.GC()
	if err := pprof.WriteHeapProfile(memF); err != nil {
		panic(err)
	}
	_ = memF.Close()

	os.Exit(code)
}
