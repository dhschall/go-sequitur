package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"sort"
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
	region_size  = flag.Uint64("region-size", 64, "config file for this experiment")
	pf_distance  = flag.Uint("max-distance", 5, "config file for this experiment")
	address_file = flag.String("file", "data.json", "config file for this experiment")
	results_file = flag.String("o", "edit_dist.csv", "Path where the results should be written")
	log_file     = flag.String("log", "", "Logfile")
	workload     = flag.String("wl", "AES-G", "Workload")
	inv          = flag.Uint("inv", 17, "Invocation")
	first_only   = flag.Bool("compare-first", false, "Compare the unique first misses")
	with_header  = flag.Bool("header", false, "Compare the unique first misses")
	max          = flag.Uint("m", 0, "config file for this experiment")

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

	data := make(map[string]map[string][]uint64)
	ReadJson(&data, *address_file)

	log.Printf("Workload: %s, inv: %d, rs:%d\n", *workload, *inv, *region_size)
	// fmt.Println("Length data", len(data))

	n := *inv
	d1 := data[*workload][fmt.Sprintf("%d", n)]
	n_1 := n + 1
	d2 := data[*workload][fmt.Sprintf("%d", n_1)]
	log.Printf("Length d1 %d, d2 %d", len(d1), len(d2))

	// Process results
	process(d1, d2)

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
func LevenshteinDistance[T uint64 | int](trace1, trace2 []T) (int, float32, []int) {

	// Get and store length of these strings
	trace1len := len(trace1)
	trace2len := len(trace2)
	// if trace1len == 0 {
	// 	return trace2len, 0, []int{}
	// } else if trace2len == 0 {
	// 	return trace1len, 0, []int{}
	// } else if Equal(trace1, trace2) {
	// 	return 0, 1, []int{}
	// }

	column := make([]int, trace1len+1)

	for y := 1; y <= trace1len; y++ {
		column[y] = y
	}
	for x := 1; x <= trace2len; x++ {
		column[0] = x
		lastkey := x - 1
		for y := 1; y <= trace1len; y++ {
			oldkey := column[y]
			var i int
			if trace1[y-1] != trace2[x-1] {
				i = 1
			}
			column[y] = Min(
				Min(column[y]+1, // insert
					column[y-1]+1), // delete
				lastkey+i) // substitution
			lastkey = oldkey
		}
	}

	distance := column[trace1len]

	max_len := Max(trace1len, trace2len)
	distance_frac := float32(max_len-distance) / float32(max_len)

	// if trace1len >= trace2len {
	// 	distance_frac = float32(trace1len-distance) / float32(trace1len)
	// } else {
	// 	distance_frac = float32(trace2len-distance) / float32(trace2len)
	// }

	return distance, distance_frac, column
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

	for i := 0; i < tracelen-1; i++ {
		v1 := trace[i]
		v2 := trace[i+1]
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

	printHistogram(histogram(successors))
}

func process(d1, d2 []uint64) {

	// n := *inv
	// Calculate the unique, temporal-ordered Trace
	var s uint = 0
	if *max != 0 {
		d1 = d1[s : s+*max]
		d2 = d2[s : s+*max]
	}

	log.Printf("Length rec %d, repl %d ", len(d1), len(d2))

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

func ReadJson(data interface{}, filename string) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.Unmarshal(content, data)
	if err != nil {
		log.Fatalln("error:", err)
	}
}

func WriteJson(data interface{}, outname string) {
	_data, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalln(err)
	}

	// outfile = outfile + fmt.Sprintf("-n%d_t%s_b%s_%s.fp.json", n, s_filter_t, s_branch_t, s_extern_skip)

	err = ioutil.WriteFile(outname, _data, 0644)
	if err != nil {
		log.Fatalln(err)
	}
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

	bc := 16

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
		res.WriteString(fmt.Sprintf("  %4.3f [%v]\t|%v\n", buckets[i].Mark, buckets[i].Count, strings.Repeat(barChar, barLen)))
	}
	log.Printf("\n%s", res.String())
	return res.String()
}
