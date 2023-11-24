package main

import (
	"bytes"
	// "encoding/json"
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

const INF int64 = math.MaxInt64

func NewRegion(addr uint64) Region {
	return Region{regionAddr(addr), 0, INF, []uint64{addr}}
}

type Region struct {
	raddr       uint64
	occurrences int
	delta       int64
	blocks      []uint64
}

func (r *Region) InBlocks(addr uint64) bool {
	for _, b := range r.blocks {
		if addr == b {
			return true
		}
	}
	return false
}

func (r *Region) AddBlock(addr uint64) bool {
	for _, b := range r.blocks {
		if addr == b {
			return true
		}
	}
	r.blocks = append(r.blocks, addr)
	return false
}

func regionAddr(addr uint64) uint64 {
	return (addr / *region_size)
}

var (
	region_size = flag.Uint64("region-size", 64, "config file for this experiment")
	order       = flag.Int("order", 1, "config file for this experiment")
	mask        = flag.Int("mask", 0, "config file for this experiment")

	address_file = flag.String("file", "data.json", "config file for this experiment")
	results_file = flag.String("o", "markov_hist.csv", "Path where the results should be written")
	log_file     = flag.String("log", "", "Logfile")
	workload     = flag.String("wl", "AES-G", "Workload")
	inv          = flag.Uint("inv", 17, "Invocation")
	first_only   = flag.Bool("compare-first", false, "Compare the unique first misses")
	with_header  = flag.Bool("header", false, "Compare the unique first misses")
	max          = flag.Uint("m", 0, "config file for this experiment")
	bins         = flag.Int("bins", 8, "Number of bins in the histogram")

	verbose     = flag.Bool("v", false, "Print more infos: (DebugLevel)")
	results_dir string
)

func main() {

	flag.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	if *log_file != "" {

		f, err := os.OpenFile(*log_file, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Failed to create logfile" + *log_file)
			panic(err)
		}
		defer f.Close()
		// Output to stdout instead of the default stderr
		log.SetOutput(f)
	}

	data := ReadUintFile(*address_file)

	log.Printf("Workload: %s, inv: %d, order:%d\n", *workload, *inv, *order)

	log.Printf("Length data %d", len(data))

	// Process results
	process(data)

}

func buildTrace(data []uint64, trace *[]Region) {
	for _, addr := range data {
		*trace = append(*trace, NewRegion(uint64(addr)))
	}
}

// Min return the smallest integer among the two in parameters
func Min[T uint64 | int](a T, b T) T {
	if b < a {
		return b
	}
	return a
}

// Max return the largest integer among the two in parameters
func Max[T uint64 | int](a, b T) T {
	if b > a {
		return b
	}
	return a
}

// Equal compare two rune arrays and return if they are equals or not
func Equal[T uint64 | int](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// LevenshteinDistance calculate the distance between two string
// This algorithm allow insertions, deletions and substitutions to change one string to the second
// Compatible with non-ASCII characters
func MarkovChain[T uint64 | int](trace []T) map[T]map[T]int {

	// type successor map[T]int

	// Get and store length of these strings
	tracelen := len(trace)
	// if trace1len == 0 {
	// 	return trace2len, 0, []int{}
	// } else if trace2len == 0 {
	// 	return trace1len, 0, []int{}
	// } else if Equal(trace1, trace2) {
	// 	return 0, 1, []int{}
	// }
	smatrix := make(map[T]map[T]int)
	state := trace[0]

	for i := 0; i < tracelen-1; i++ {
		var v1, v2 T
		if *order > 1 && i >= *order {
			// Remove the first element of the state
			state ^= trace[i-*order]
			// Add the new element to the state
			state ^= trace[i]
			v1 = state
		} else {
			v1 = trace[i]
		}
		v2 = trace[i+1]

		if *mask != 0 {
			v1 >>= *mask
			v2 >>= *mask
		}

		if smatrix[v1] == nil {
			smatrix[v1] = make(map[T]int, 0)
		}
		smatrix[v1][v2]++
	}

	return smatrix
}

func nSuccessors[T uint64 | int](trace []T) {

	mchain := MarkovChain[T](trace)
	log.Printf("Length Markov Chain %d, repl %d ", len(mchain))

	successors := make([]float64, 0, len(mchain))
	for _, v := range mchain {
		successors = append(successors, float64(len(v)))
	}

	hist := histogram(successors)
	printHistogram(hist)

	writeHistogram(hist, *results_file, *with_header)

}

func process(d1 []uint64) {

	// n := *inv
	// Calculate the unique, temporal-ordered Trace
	var s uint = 0
	if *max != 0 {
		d1 = d1[s : s+*max]
	}

	log.Printf("Length rec %d", len(d1))

	nSuccessors[uint64](d1)
}

func jaccard(a, b []uint64) float32 {
	ma := make(map[uint64]bool)
	mb := make(map[uint64]bool)

	for _, item := range a {
		ma[item] = true
	}

	for _, item := range b {
		mb[item] = true
	}
	union := len(ma)
	intersect := 0

	for key, _ := range mb {
		if _, ok := ma[key]; ok {
			intersect += 1
		} else {
			union += 1
		}
	}

	// fmt.Println(union, " ", intersect)
	return float32(intersect) / float32(union)
}

func ReadUintFile(filename string) (data []uint64) {
	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalln("Error opening file:", err)
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Read each line
	for scanner.Scan() {
		line := scanner.Text()
		num, err := strconv.ParseUint(line, 10, 64)
		if err != nil {
			log.Fatalln("Error parsing line:", err)
			continue
		}
		data = append(data, num)
	}

	// Check if there was an error during scanning
	if err := scanner.Err(); err != nil {
		log.Fatalln("Error scanning file:", err)

	}
	return
}

// func WriteStrings(filename string, data []string, header string) {

// 	f, err := os.Create(filename)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	if header != "" {
// 		fmt.Fprintln(f, header)
// 	}
// 	for _, k := range data {
// 		fmt.Fprintln(f, k)
// 	}
// 	err = f.Close()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }

func writeTrace(trace *orderedmap.OrderedMap[uint64, *Region], filename string) {

	var output bytes.Buffer

	for pair := trace.Oldest(); pair != nil; pair = pair.Next() {
		// fmt.Printf("%d => %d\n", pair.Key, pair.Value.occurrences)
		output.WriteString(fmt.Sprintf("%d => %d : %v\n", pair.Key,
			pair.Value.occurrences, pair.Value.blocks))
	}

	ioutil.WriteFile(filename, output.Bytes(), 0644)
}

func writeTraceReal(trace *[]Region, filename string) {

	var output bytes.Buffer

	for i := 0; i < len(*trace); i++ {
		// fmt.Printf("%d => %d\n", pair.Key, pair.Value.occurrences)
		output.WriteString(fmt.Sprintf("%d => %d : %d : %v\n",
			(*trace)[i].raddr, (*trace)[i].occurrences, (*trace)[i].delta,
			(*trace)[i].blocks))
	}

	ioutil.WriteFile(filename, output.Bytes(), 0644)
}

func WriteSlice[V any](filename string, data []V, header string) {

	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}

	if header != "" {
		fmt.Fprintln(f, header)
	}
	for _, k := range data {
		fmt.Fprintln(f, k)
	}
	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
}

///// Print functions
// Bucket holds histogram data
type Bucket struct {
	// The Mark for histogram bucket in seconds
	Mark float64 `json:"mark"`

	// The count in the bucket
	Count int `json:"count"`

	// The frequency of results in the bucket as a decimal percentage
	Frequency float64 `json:"frequency"`
}

// func histogram(latencies []float64, slowest, fastest float64) []Bucket {

func histogram(latencies []float64) []Bucket {

	sort.Float64s(latencies)

	bc := *bins

	fastest := latencies[0]
	// slowest := latencies[len(latencies)-1]

	slowest := float64(bc + 1)

	buckets := make([]float64, bc+1)
	counts := make([]int, bc+1)
	bs := (float64(slowest) - (fastest)) / float64(bc)
	for i := 0; i < bc; i++ {
		buckets[i] = fastest + bs*float64(i)
	}
	buckets[bc] = slowest
	var bi int
	var max int
	for i := 0; i < len(latencies); {
		if latencies[i] <= buckets[bi] || bi == len(buckets)-1 {
			i++
			counts[bi]++
			if max < counts[bi] {
				max = counts[bi]
			}
		} else if bi < len(buckets)-1 {
			bi++
		}
	}
	res := make([]Bucket, len(buckets))
	for i := 0; i < len(buckets); i++ {
		res[i] = Bucket{
			Mark:      buckets[i],
			Count:     counts[i],
			Frequency: float64(counts[i]) / float64(len(latencies)),
		}
	}
	return res
}

const (
	barChar = "∎"
)

func printHistogram(buckets []Bucket) string {
	max := 0
	for _, b := range buckets {
		if v := b.Count; v > max {
			max = v
		}
	}
	res := new(bytes.Buffer)
	for i := 0; i < len(buckets); i++ {
		// Normalize bar lengths.
		var barLen int
		if max > 0 {
			barLen = (buckets[i].Count*40 + max/2) / max
		}
		res.WriteString(fmt.Sprintf("  %2.0f [%2.1f]\t|%v\n", buckets[i].Mark, buckets[i].Frequency*100, strings.Repeat(barChar, barLen)))
	}
	log.Printf("\n%s", res.String())
	return res.String()
}

func writeHistogram(buckets []Bucket, filename string, header bool) {
	res := new(bytes.Buffer)

	if header {
		for _, b := range buckets {
			res.WriteString(fmt.Sprintf("%4.3f,", b.Mark))
		}
		res.WriteString("\n")
	}

	for _, b := range buckets {
		res.WriteString(fmt.Sprintf("%4.3f,", b.Frequency))
	}

	ioutil.WriteFile(filename, res.Bytes(), 0644)
}
