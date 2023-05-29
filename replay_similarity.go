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

	"github.com/dhschall/uniplot/histogram"

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
	results_file = flag.String("o", "pf_opportunity.csv", "Path where the results should be written")
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
	unique(d1, d2)

}

func buildTrace(data []uint64, trace *[]Region) {
	for _, addr := range data {
		*trace = append(*trace, NewRegion(uint64(addr)))
	}
}

func unique(d1, d2 []uint64) {

	// n := *inv
	// Calculate the unique, temporal-ordered Trace
	var s uint = 0
	if *max != 0 {
		d1 = d1[s : s+*max]
		d2 = d2[s : s+*max]
	}

	// record_trace := make([]Region, 0)
	var recordT, replayT []Region
	buildTrace(d1, &recordT)
	buildTrace(d2, &replayT)

	// for i := 0; i < len(recordT); i++ {
	// 	log.Printf("%d ", recordT[i].raddr)
	// }

	log.Printf("Length rec %d, repl %d ", len(recordT), len(replayT))

	// _start := 0
	// for i := 0; i < len(recordT); i++ {
	// 	raddr := recordT[i].raddr
	// 	found := false
	// 	// Find the region in the replay trace.
	// 	// Skip all that have already been touched
	// 	if
	// 	j := _start
	// 	first_touch_found := false
	// 	for ; j < len(replayT); j++ {

	// 		tmp := replayT[j]
	// 		// Optimization will to not waste to much time to find the beginning
	// 		if !first_touch_found && tmp.occurrences == 0 {
	// 			first_touch_found = true
	// 			_start = j
	// 		}

	// 		log.Printf("%d: %d\n", i, j)
	// 		if tmp.raddr == raddr && tmp.occurrences == 0 {
	// 			found = true
	// 			break
	// 		}
	// 	}

	// 	if found {
	// 		delta := j - i
	// 		log.Debugf("Found %#d: Irec. %d, Irep:%d. delta:%d\n", raddr, i, j, delta)
	// 		replayT[j].delta = int64(delta)
	// 		replayT[j].occurrences = 1
	// 	}

	// }

	// Initialization
	// Find the first entry of the record trace in the replay trace
	_start := 0
	i0, j0 := 0, 0
	trigger_found := false
	for ; i0 < len(recordT)-1; i0++ {

		// If we do not already have the trigger block index in the replay
		// trace that means its the first access. Or it was not found in the replay trace.
		// Hence we loose the opportunity to trigger a prefetch.
		if trigger_found == false {

			raddr := recordT[i0].raddr
			found := false
			j0 = _start
			for ; j0 < len(replayT); j0++ {

				tmp := replayT[j0]
				if tmp.raddr == raddr && tmp.occurrences == 0 {
					found = true
					replayT[j0].occurrences = 1
					replayT[j0].delta = -INF
					break
				}
			}
			if found {
				log.Debugf("Trigger block found %#d: Irec. %d, Irep:%d\n", raddr, i0, j0)
			} else {
				log.Debugf("Could not find trigger block: %#d. Goto next\n", raddr)
				continue
			}
		}

		// Now that we have the index of the trigger block get the next
		// block in the record trace. (predicted block)
		i1 := i0 + 1
		// Check if we are at the end of the record trace
		if i1 >= len(recordT) {
			break
		}

		raddr := recordT[i0].raddr
		pred_raddr := recordT[i1].raddr

		ss := fmt.Sprintf("Predict rec[%d]:%#d +1 %#d", i0, raddr, pred_raddr)

		// Now find the predicted block in the trace.
		found := false
		j1 := _start
		// first_touch_found := false

		// Find the first occurence of the predicted block
		// Skip all the once already seen.
		for ; j1 < len(replayT); j1++ {

			tmp := replayT[j1]
			if tmp.raddr == pred_raddr && tmp.occurrences == 0 {
				found = true
				break
			}
		}

		if found {
			delta := j1 - j0 - 1
			log.Debugf("%s -> Found: rep[%d] %+d rep[%d] > delta %d\n", ss, j0, delta+1, j1, delta)
			recordT[i1].delta = int64(delta)
			replayT[j1].delta = int64(delta)
			replayT[j1].occurrences = 1

			j0 = j1
			trigger_found = true
		} else {

			// If not found this means we can also not make a prediction.
			// We restart with the next block
			j0 = 0
			trigger_found = false
			log.Debugf("%s -> Not found!!\n", ss)
			i0 += 1
		}

	}

	writeTraceReal(&recordT, "record_T.txt")
	writeTraceReal(&replayT, "replay_T.txt")

	// Calculate the distribution of deltas
	dist := make(map[int64]int)
	data := []float64{}
	for i := 0; i < len(recordT); i++ {
		delta := recordT[i].delta
		data = append(data, float64(delta))
		// log.Printf("%d: delta:%d\n", i, delta)
		if v, ok := dist[delta]; ok {
			dist[delta] = v + 1
		} else {
			dist[delta] = 1
		}
	}

	bins := 9
	hist := histogram.Hist(bins, data)

	maxWidth := 5
	histogram.Fprint(os.Stdout, hist, histogram.Linear(maxWidth))
	//

	results := []string{}
	// num_samples := float64(len(replayT))

	// for k, v := range dist {
	// 	results = append(results, fmt.Sprintf("%s,%d,%d,%d,%d,%f", *workload, *inv, *region_size, k, v, float64(v)/num_samples))
	// }
	// header := ""
	// if *with_header {
	// 	header = "wl,inv,rs,delta,occur,hist"
	// }

	for i := 0; i < len(replayT); i++ {
		results = append(results, fmt.Sprintf("%s,%d,%d", *workload, *inv, replayT[i].delta))
	}
	header := ""
	if *with_header {
		header = "wl,inv,delta"
	}

	WriteSlice(*results_file, results, header)

	// results := []int64{}
	// for i := 0; i < len(replayT); i++ {
	// 	results = append(results, replayT[i].delta)
	// }

	// WriteSlice(*results_file, results, "")
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
