package benchmarks

import (
	M "github.com/agkloop/go_memoize"
	"testing"
	"time"
)

const directBenchLRUCapacity = 1024

func mustMemoized[F any](fn F, err error) F {
	if err != nil {
		panic(err)
	}
	return fn
}

func DoSomThingZero() string         { return "a" }
func DoSomThing1(a string) string    { return a }
func DoSomThing2(a, b string) string { return a + b }
func DoSomThing3(a string, b string, c string) string {
	//res := "a"
	//for i := 0; i < 1000000; i++ {
	//	res = res + a
	//}
	return a + b + c
}
func DoSomThing4(a string, b string, c string, s int) string {
	//res := "a"
	//for i := 0; i < 1000000; i++ {
	//	res = res + a
	//}
	return a + b + c
}

func BenchmarkDo0Mem(b *testing.B) {
	DoSomThingZeroMemoized := mustMemoized(M.Memoize(DoSomThingZero, M.Opts().WithTTL(10*time.Minute)))
	b.ReportAllocs()

	for b.Loop() {
		DoSomThingZeroMemoized()
	}
}

func BenchmarkDo0LRU(b *testing.B) {
	DoSomThingZeroMemoized := lruMemoize(DoSomThingZero, 10*time.Minute, 1)
	b.ReportAllocs()

	for b.Loop() {
		DoSomThingZeroMemoized()
	}
}

func BenchmarkDo1Mem(b *testing.B) {
	DoSomThing1Memoized := mustMemoized(M.Memoize1(DoSomThing1, M.Opts().WithTTL(10*time.Minute)))
	params := []string{"1111", "2222", "3333", "4444"}
	idx := 0
	b.ReportAllocs()

	for b.Loop() {
		DoSomThing1Memoized(params[idx%len(params)])
		idx++
	}
}

func BenchmarkDo1LRU(b *testing.B) {
	DoSomThing1Memoized := lruMemoize1(DoSomThing1, 10*time.Minute, directBenchLRUCapacity)
	params := []string{"1111", "2222", "3333", "4444"}
	idx := 0
	b.ReportAllocs()

	for b.Loop() {
		DoSomThing1Memoized(params[idx%len(params)])
		idx++
	}
}

func BenchmarkDo2Mem(b *testing.B) {
	DoSomThing2Memoized := mustMemoized(M.Memoize2(DoSomThing2, M.Opts().WithTTL(10*time.Minute)))
	params := []struct {
		a string
		b string
	}{
		{"1-", "1111"},
		{"2-", "2222"},
		{"3-", "3333"},
		{"4-", "4444"},
	}
	idx := 0
	b.ReportAllocs()

	for b.Loop() {
		DoSomThing2Memoized(params[idx%len(params)].a, params[idx%len(params)].b)
		idx++
	}
}

func BenchmarkDo2LRU(b *testing.B) {
	DoSomThing2Memoized := lruMemoize2(DoSomThing2, 10*time.Minute, directBenchLRUCapacity)
	params := []struct {
		a string
		b string
	}{
		{"1-", "1111"},
		{"2-", "2222"},
		{"3-", "3333"},
		{"4-", "4444"},
	}
	idx := 0
	b.ReportAllocs()

	for b.Loop() {
		DoSomThing2Memoized(params[idx%len(params)].a, params[idx%len(params)].b)
		idx++
	}
}

func BenchmarkDo3Mem(b *testing.B) {
	DoSomThing3Memoized := mustMemoized(M.Memoize3(DoSomThing3, M.Opts().WithTTL(10*time.Minute)))
	params := []struct {
		a, b, c string
	}{
		{"1111", "2222", "3333"},
		{"4444", "5555", "6666"},
		{"7777", "8888", "9999"},
		{"aaaa", "bbbb", "cccc"},
	}
	idx := 0
	b.ReportAllocs()

	for b.Loop() {
		DoSomThing3Memoized(params[idx%len(params)].a, params[idx%len(params)].b, params[idx%len(params)].c)
		idx++
	}
}

func BenchmarkDo3LRU(b *testing.B) {
	DoSomThing3Memoized := lruMemoize3(DoSomThing3, 10*time.Minute, directBenchLRUCapacity)
	params := []struct {
		a, b, c string
	}{
		{"1111", "2222", "3333"},
		{"4444", "5555", "6666"},
		{"7777", "8888", "9999"},
		{"aaaa", "bbbb", "cccc"},
	}
	idx := 0
	b.ReportAllocs()

	for b.Loop() {
		DoSomThing3Memoized(params[idx%len(params)].a, params[idx%len(params)].b, params[idx%len(params)].c)
		idx++
	}
}

func BenchmarkDo4Mem(b *testing.B) {
	DoSomThing4Memoized := mustMemoized(M.Memoize4(DoSomThing4, M.Opts().WithTTL(10*time.Minute)))
	params := []struct {
		a, b, c string
		s       int
	}{
		{"1111", "2222", "3333", 1},
		{"4444", "5555", "6666", 2},
		{"7777", "8888", "9999", 3},
		{"aaaa", "bbbb", "cccc", 4},
	}
	idx := 0
	b.ReportAllocs()

	for b.Loop() {
		DoSomThing4Memoized(params[idx%len(params)].a, params[idx%len(params)].b, params[idx%len(params)].c, params[idx%len(params)].s)
		idx++
	}
}

func BenchmarkDo4LRU(b *testing.B) {
	DoSomThing4Memoized := lruMemoize4(DoSomThing4, 10*time.Minute, directBenchLRUCapacity)
	params := []struct {
		a, b, c string
		s       int
	}{
		{"1111", "2222", "3333", 1},
		{"4444", "5555", "6666", 2},
		{"7777", "8888", "9999", 3},
		{"aaaa", "bbbb", "cccc", 4},
	}
	idx := 0
	b.ReportAllocs()

	for b.Loop() {
		DoSomThing4Memoized(params[idx%len(params)].a, params[idx%len(params)].b, params[idx%len(params)].c, params[idx%len(params)].s)
		idx++
	}
}
