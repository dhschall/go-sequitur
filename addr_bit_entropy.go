package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"

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
	log.Printf("Length d1 %d, d2 %d", len(d1))

	// Process results
	process(d1)

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

func process(d1 []uint64) {

	// n := *inv
	// Calculate the unique, temporal-ordered Trace
	var s uint = 0
	if *max != 0 {
		d1 = d1[s : s+*max]
	}

	for i := 0; i < 64; i++ {

	}

	log.Printf("Length rec %d, repl %d ", len(d1), len(d2))

	dist, dist_frac, edits := LevenshteinDistance[uint64](d1, d2)

	ss := fmt.Sprintf("%s,%d,%d,%d,%d,%f", *workload, *inv, len(d1), len(d2), dist, dist_frac)

	log.Printf("Distance: %d, %f, %v \n", dist, dist_frac, edits[:4])
	log.Printf("Distance: %s \n", ss)

	results := []string{}
	results = append(results, ss)

	// for i := 0; i < len(edits); i++ {
	// 	results = append(results, fmt.Sprintf("%s,%d,%d", *workload, *inv, ))
	// }

	// for i := 0; i < len(edits); i++ {
	// 	if i >= len(d1) || i >= len(d2) {
	// 		break
	// 	}
	// 	results = append(results, fmt.Sprintf("%#x | %d | %#x", d1[i], edits[i], d2[i]))
	// }

	// log.Printf("len: %d\n", len(results))

	header := ""
	if *with_header {
		header = "wln,inv,l1,l2,dist,distf"
	}

	WriteSlice(*results_file, results, header)
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
