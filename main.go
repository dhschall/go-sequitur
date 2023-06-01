package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/dhschall/uniplot/histogram"
	log "github.com/sirupsen/logrus"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type Region struct {
	raddr       uint64
	occurrences int
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

func NewRegion(addr uint64) *Region {
	return &Region{regionAddr(addr), 1, []uint64{addr}}
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

	verbose     = flag.Bool("v", false, "Print more infos: (DebugLevel)")
	results_dir string
)

func main() {

	flag.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.ErrorLevel)
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

func unique(d1, d2 []uint64) {

	n := *inv
	n_1 := n + 1
	// Calculate the unique, temporal-ordered Trace
	om1 := orderedmap.New[uint64, *Region]()
	computeRecordTraceUnique(d1[:], om1)

	om2 := orderedmap.New[uint64, *Region]()
	computeRecordTraceUnique(d2[:], om2)

	log.Printf("Length UTO trace (m1: %d) (m2: %d) ", om1.Len(), om2.Len())

	filename := fmt.Sprintf("grammar_%d.txt", n)
	writeTrace(om1, filename)

	filename = fmt.Sprintf("grammar_%d.txt", n_1)
	writeTrace(om2, filename)

	var addresses []uint64
	// For first only we
	if *first_only {
		for pair := om2.Oldest(); pair != nil; pair = pair.Next() {
			// fmt.Printf("%d => %d\n", pair.Key, pair.Value.occurrences)
			addresses = append(addresses, pair.Key)
		}
	} else {
		addresses = d2
	}

	addresses = d2[:20]

	// Now compute the jaccard indicees.

	results := []string{}

	for d := 1; d < int(*pf_distance); d++ {

		jv := perdict(om1, addresses, d)
		log.Printf("Jaccard similarity for PF distance: %d -> %f\n", d, jv)

		results = append(results, fmt.Sprintf("%s,%d,%d,%d,%f", *workload, *inv, *region_size, d, jv))
	}
	header := ""
	if *with_header {
		header = "wl,inv,rs,pfd,jv"
	}

	WriteStrings(*results_file, results, header)
}

// func processRegionSize(d1,d2 )  {

// }

func perdict(recordTrace *orderedmap.OrderedMap[uint64, *Region], missTrace []uint64, distance int) float64 {
	log.Debugf("Length UTO trace %d, len addr: %d, PF distance: %d\n",
		recordTrace.Len(), len(missTrace), distance)

	n := len(missTrace) - distance
	var jv_sum float64
	for i := 0; i < n; i++ {

		addr := regionAddr(missTrace[i])

		// Get the next predictions for this tigger address
		var predictions []uint64
		pair := recordTrace.GetPair(addr)
		if pair != nil {
			pair = pair.Next()
		}

		for ; pair != nil && len(predictions) < distance; pair = pair.Next() {
			for _, b := range pair.Value.blocks {
				predictions = append(predictions, b)
			}
		}

		if len(predictions) == 0 {
			continue
		}

		// Slice out the same number of succeeding misses in the miss trace
		end := i + 1 + len(predictions)
		if end >= len(missTrace) {
			end = len(missTrace)
		}
		misses := missTrace[i+1 : end]

		jv := jaccard(misses, predictions)
		jv_sum += float64(jv)
		fmt.Printf("%d: addr: %d, MT: %v Pred: %v, jaccard %f\n", i, addr, misses, predictions, jv)

	}
	fmt.Printf("Overall jaccard: %f: %f/%d for distance: %d\n", jv_sum/float64(n), jv_sum, n, distance)
	return jv_sum / float64(n)
}

func computeRecordTraceUnique(addresses []uint64, trace *orderedmap.OrderedMap[uint64, *Region]) {
	for _, blkaddr := range addresses {

		raddr := regionAddr(blkaddr)
		pair := trace.GetPair(raddr)
		if pair != nil {
			pair.Value.occurrences += 1
			pair.Value.AddBlock(blkaddr)
		} else {
			trace.Set(raddr, NewRegion(blkaddr))
		}
	}
}

// func real(d1, d2 []uint64) {

// 	n := *inv
// 	n_1 := n + 1
// 	// Calculate the unique, temporal-ordered Trace
// 	var t1, t2 []*Region
// 	computeRecordTraceReal(d1[:], &t1)
// 	computeRecordTraceReal(d1[:], &t2)

// 	log.Printf("Length UTO trace (m1: %d) (m2: %d) ", om1.Len(), om2.Len())

// 	filename := fmt.Sprintf("grammar_%d.txt", n)
// 	writeTrace(om1, filename)

// 	filename = fmt.Sprintf("grammar_%d.txt", n_1)
// 	writeTrace(om2, filename)

// 	var addresses []uint64
// 	// For first only we
// 	if *first_only {
// 		for pair := om2.Oldest(); pair != nil; pair = pair.Next() {
// 			// fmt.Printf("%d => %d\n", pair.Key, pair.Value.occurrences)
// 			addresses = append(addresses, pair.Key)
// 		}
// 	} else {
// 		addresses = d2
// 	}

// 	// addresses = d2[:]

// 	// Now compute the jaccard indicees.

// 	results := []string{}

// 	for d := 1; d < int(*pf_distance); d++ {

// 		jv := perdict(om1, addresses, d)
// 		log.Printf("Jaccard similarity for PF distance: %d -> %f\n", d, jv)

// 		results = append(results, fmt.Sprintf("%s,%d,%d,%d,%f", *workload, *inv, *region_size, d, jv))
// 	}
// 	header := ""
// 	if *with_header {
// 		header = "wl,inv,rs,pfd,jv"
// 	}

// 	WriteStrings(*results_file, results, header)
// }

// func computeRecordTraceReal(addresses []uint64, trace *[]*Region) {
// 	for _, blkaddr := range addresses {

// 		raddr := regionAddr(blkaddr)
// 		pair := trace.GetPair(raddr)
// 		if pair != nil {
// 			pair.Value.occurrences += 1
// 			pair.Value.AddBlock(blkaddr)
// 		} else {
// 			trace.Set(raddr, &Region{1, []uint64{blkaddr}})
// 		}
// 	}
// }

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

func WriteStrings(filename string, data []string, header string) {

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

func writeTrace(trace *orderedmap.OrderedMap[uint64, *Region], filename string) {

	var output bytes.Buffer

	// Calculate the distribution of occurances
	data := []float64{}

	for pair := trace.Oldest(); pair != nil; pair = pair.Next() {
		// fmt.Printf("%d => %d\n", pair.Key, pair.Value.occurrences)
		output.WriteString(fmt.Sprintf("%d => %d : %v\n", pair.Key,
			pair.Value.occurrences, pair.Value.blocks))
		data = append(data, float64(pair.Value.occurrences))
	}

	ioutil.WriteFile(filename, output.Bytes(), 0644)
	// bins := 9
	hist := histogram.PowerHist(2, data)

	maxWidth := 5
	histogram.Fprint(os.Stdout, hist, histogram.Linear(maxWidth))
}

func writeTraceReal(trace *[]*Region, filename string) {

	var output bytes.Buffer

	for i := 0; i < len(*trace); i++ {
		// fmt.Printf("%d => %d\n", pair.Key, pair.Value.occurrences)
		output.WriteString(fmt.Sprintf("%d => %d : %v\n",
			(*trace)[i].raddr, (*trace)[i].occurrences,
			(*trace)[i].blocks))
	}

	ioutil.WriteFile(filename, output.Bytes(), 0644)
}
