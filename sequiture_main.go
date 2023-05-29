package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"

	"os"
	"unicode"
	"unicode/utf8"

	// "./sequitur"

	log "github.com/sirupsen/logrus"
)

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

func process(d1, d2 []uint64) {

	// n := *inv
	// Calculate the unique, temporal-ordered Trace
	var s uint = 0
	if *max != 0 {
		d1 = d1[s : s+*max]
		d2 = d2[s : s+*max]
	}

	log.Printf("Length rec %d, repl %d ", len(d1), len(d2))

	g := ParseAddr(d1)

	fmt.Println("Length r: ", g.NumRules())

	var output bytes.Buffer
	// if err := g.Print(&output); err != nil {
	// 	panic(err)
	// }
	if err := g.PrettyPrint(&output); err != nil {
		panic(err)
	}

	filename := fmt.Sprintf("seq_grammar_%d.txt", 1)
	ioutil.WriteFile(filename, output.Bytes(), 0644)

	output.Reset()

	fmt.Println("Length r: ", g.NumRules())
	err, heads, repeat, indexes := g.OpportunityPrint(&output)
	if err != nil {
		panic(err)
	}

	log.Printf("Heads %d , repeats: %d, indexes: %d", heads, repeat, indexes)

	filename = fmt.Sprintf("seq_opportunity_%d.txt", 1)
	ioutil.WriteFile(filename, output.Bytes(), 0644)

	// log.Printf("Distance: %s \n", ss)

	// results := []string{}
	// results = append(results, ss)

	// // for i := 0; i < len(edits); i++ {
	// // 	results = append(results, fmt.Sprintf("%s,%d,%d", *workload, *inv, ))
	// // }

	// // for i := 0; i < len(edits); i++ {
	// // 	if i >= len(d1) || i >= len(d2) {
	// // 		break
	// // 	}
	// // 	results = append(results, fmt.Sprintf("%#x | %d | %#x", d1[i], edits[i], d2[i]))
	// // }

	// // log.Printf("len: %d\n", len(results))

	// header := ""
	// if *with_header {
	// 	header = "wln,inv,l1,l2,dist,distf"
	// }

	// WriteSlice(*results_file, results, header)
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

// Grammar is a constructed grammar.  The zero value is safe to call Parse on.
type Grammar struct {
	table  digrams
	base   *rules
	ruleID uint64
}

func (g *Grammar) NumRules() int {
	return len(g.table)
}

func (g *Grammar) nextID() uint64 {
	g.ruleID++
	return g.ruleID
}

type rules struct {
	id    uint64
	guard *symbols
	count int
}

func (r *rules) first() *symbols { return r.guard.next }
func (r *rules) last() *symbols  { return r.guard.prev }

func (g *Grammar) newRules() *rules {
	r := &rules{id: g.nextID()}
	r.guard = g.newGuard(r)
	return r
}

func (g *Grammar) newSymbolFromValue(sym uint64) *symbols {
	return &symbols{
		g:     g,
		value: sym,
	}
}

func (g *Grammar) newSymbolFromRule(r *rules) *symbols {
	r.count++
	return &symbols{
		g:     g,
		value: r.id,
		rule:  r,
	}
}

func (g *Grammar) newGuard(r *rules) *symbols {
	s := &symbols{g: g, value: r.id, rule: r}
	s.next, s.prev = s, s
	return s
}

func (g *Grammar) newSymbol(s *symbols) *symbols {
	if s.isNonTerminal() {
		return g.newSymbolFromRule(s.rule)
	}
	return g.newSymbolFromValue(s.value)
}

type symbols struct {
	g          *Grammar
	next, prev *symbols
	value      uint64
	rule       *rules
}

func (s *symbols) isGuard() (b bool)   { return s.isNonTerminal() && s.rule.first().prev == s }
func (s *symbols) isNonTerminal() bool { return s.rule != nil }

func (s *symbols) delete() {
	s.prev.join(s.next)
	s.deleteDigram()
	if s.isNonTerminal() {
		s.rule.count--
	}
}

func (s *symbols) isTriple() bool {
	return s.prev != nil && s.next != nil &&
		s.value == s.prev.value &&
		s.value == s.next.value
}

func (s *symbols) join(right *symbols) {
	if s.next != nil {
		s.deleteDigram()

		if right.isTriple() {
			s.g.table.insert(right)
		}

		if s.isTriple() {
			s.g.table.insert(s.prev)
		}
	}
	s.next = right
	right.prev = s
}

func (s *symbols) insertAfter(y *symbols) {
	y.join(s.next)
	s.join(y)
}

func (s *symbols) deleteDigram() {
	if s.isGuard() || s.next.isGuard() {
		return
	}
	s.g.table.delete(s)
}

func (s *symbols) check() bool {
	if s.isGuard() || s.next.isGuard() {
		return false
	}

	x, ok := s.g.table.lookup(s)
	if !ok {
		s.g.table.insert(s)
		return false
	}

	if x.next != s {
		s.match(x)
	}

	return true
}

func (s *symbols) expand() {
	left := s.prev
	right := s.next
	f := s.rule.first()
	l := s.rule.last()

	s.g.table.delete(s)

	left.join(f)
	l.join(right)

	s.g.table.insert(l)
}

func (s *symbols) substitute(r *rules) {
	q := s.prev

	q.next.delete()
	q.next.delete()

	q.insertAfter(s.g.newSymbolFromRule(r))

	if !q.check() {
		q.next.check()
	}
}

func (s *symbols) match(m *symbols) {
	var r *rules

	if m.prev.isGuard() && m.next.next.isGuard() {
		r = m.prev.rule
		s.substitute(r)
	} else {
		r = s.g.newRules()

		r.last().insertAfter(s.g.newSymbol(s))
		r.last().insertAfter(s.g.newSymbol(s.next))

		m.substitute(r)
		s.substitute(r)

		s.g.table.insert(r.first())
	}

	if r.first().isNonTerminal() && r.first().rule.count == 1 {
		r.first().expand()
	}
}

type digram struct{ one, two uint64 }

type digrams map[digram]*symbols

func (t digrams) lookup(s *symbols) (*symbols, bool) {
	d := digram{s.value, s.next.value}
	m, ok := t[d]
	return m, ok
}

func (t digrams) insert(s *symbols) {
	d := digram{s.value, s.next.value}
	t[d] = s
}

func (t digrams) delete(s *symbols) {
	d := digram{s.value, s.next.value}
	if m, ok := t[d]; ok && s == m {
		delete(t, d)
	}
}

type prettyPrinter struct {
	rules                  []*rules
	index                  map[*rules]int
	heads, repeat, indexes int
	rule_sizes             []int
	oppMap                 [][]int
}

func (pr *prettyPrinter) print(w io.Writer, r *rules) error {
	for p := r.first(); !p.isGuard(); p = p.next {
		if p.isNonTerminal() {
			if err := pr.printNonTerminal(w, p.rule); err != nil {
				return err
			}
		} else {
			if err := pr.printTerminal(w, p.value); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func (pr *prettyPrinter) printNonTerminal(w io.Writer, r *rules) error {
	var i int

	if idx, ok := pr.index[r]; ok {
		i = idx
	} else {
		i = len(pr.rules)
		pr.index[r] = i
		pr.rules = append(pr.rules, r)
	}

	_, err := fmt.Fprint(w, " ", i)
	return err
}

func (pr *prettyPrinter) printReindexes(w io.Writer, r *rules) error {
	var i int

	if idx, ok := pr.index[r]; ok {
		i = idx
	} else {
		i = len(pr.rules)
		pr.index[r] = i
		pr.rules = append(pr.rules, r)
	}

	_, err := fmt.Fprint(w, " I")
	return err
}

func (pr *prettyPrinter) printTerminal(w io.Writer, sym uint64) error {
	// out := make([]byte, 1, 9)
	// out[0] = ' '
	// b := make([]byte, 8)
	// binary.LittleEndian.PutUint64(b, sym)

	// rb := runeOrByte(sym)
	// switch r := rb.rune(); r {
	// case ' ':
	// 	out = append(out, '_')
	// case '\n':
	// 	out = append(out, []byte("\\n")...)
	// case '\t':
	// 	out = append(out, []byte("\\t")...)
	// case '\\', '(', ')', '_', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
	// 	out = append(out, '\\', byte(r))
	// default:
	// 	out = rb.appendEscaped(out)
	// }
	// _, err := w.Write([]byte{' '})
	// _, err = w.Write(b)
	_, err := w.Write([]byte(string(' ') + fmt.Sprintf("%d", sym)))

	return err
}

func (pr *prettyPrinter) expand(r *rules) {

	n := 1
	for p := r.first(); !p.isGuard(); p = p.next {
		if p.isNonTerminal() {

			if _, ok := pr.index[r]; !ok {
				pr.index[p.rule] = len(pr.rules)
				pr.rules = append(pr.rules, p.rule)
			}

		}
		n++
	}
	pr.rule_sizes = append(pr.rule_sizes, n)
}

func (pr *prettyPrinter) printExpanded(w io.Writer, r *rules) error {
	p := r.first()
	pr.heads++
	if _, err := fmt.Fprint(w, " H"); err != nil {
		return err
	}

	for ; !p.isGuard(); p = p.next {
		if p.isNonTerminal() {
			pr.indexes++
			if _, err := fmt.Fprint(w, " I"); err != nil {
				return err
			}
		} else {
			pr.repeat++
			if _, err := fmt.Fprint(w, " O"); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func (pr *prettyPrinter) printHead(w io.Writer, r *rules) error {
	var i int

	if idx, ok := pr.index[r]; ok {
		i = idx
	} else {
		i = len(pr.rules)
		pr.index[r] = i
		pr.rules = append(pr.rules, r)
	}

	_, err := fmt.Fprint(w, " ", i)
	return err
}

func (pr *prettyPrinter) printOpportunity(w io.Writer, sym uint64) error {
	// out := make([]byte, 1, 9)
	// out[0] = ' '
	// b := make([]byte, 8)
	// binary.LittleEndian.PutUint64(b, sym)

	// rb := runeOrByte(sym)
	// switch r := rb.rune(); r {
	// case ' ':
	// 	out = append(out, '_')
	// case '\n':
	// 	out = append(out, []byte("\\n")...)
	// case '\t':
	// 	out = append(out, []byte("\\t")...)
	// case '\\', '(', ')', '_', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
	// 	out = append(out, '\\', byte(r))
	// default:
	// 	out = rb.appendEscaped(out)
	// }
	// _, err := w.Write([]byte{' '})
	// _, err = w.Write(b)
	_, err := w.Write([]byte(string(' ') + fmt.Sprintf("%d", sym)))

	return err
}

func printTerminal(w io.Writer, sym uint64) error {
	// out := make([]byte, 1, 9)
	// out[0] = ' '
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, sym)

	_, err := w.Write([]byte{' '})
	_, err = w.Write(b)

	return err
}

func rawPrint(w io.Writer, r *rules) error {
	for p := r.first(); !p.isGuard(); p = p.next {
		if p.isNonTerminal() {
			if err := rawPrint(w, p.rule); err != nil {
				return err
			}
		} else {
			if err := printTerminal(w, p.value); err != nil {
				return err
			}
		}
	}
	return nil
}

// Print reconstructs the input to w
func (g *Grammar) Print(w io.Writer) error {
	return rawPrint(w, g.base)
}

// PrettyPrint outputs the grammar to w
func (g *Grammar) PrettyPrint(w io.Writer) error {

	pr := prettyPrinter{
		index: make(map[*rules]int),
		rules: []*rules{g.base},
	}

	log.Println("Len rules ", len(pr.rules))

	for i := 0; i < len(pr.rules); i++ {
		if _, err := fmt.Fprintf(w, "%d (%dx)\t->", i, pr.rules[i].count); err != nil {
			return err
		}

		if err := pr.print(w, pr.rules[i]); err != nil {
			return err
		}
	}

	return nil
}

// PrettyPrint outputs the grammar to w
func (g *Grammar) OpportunityPrint(w io.Writer) (error, int, int, int) {

	pr := prettyPrinter{
		index: make(map[*rules]int),
		rules: []*rules{g.base},
	}

	// First expand the grammar
	for i := 0; i < len(pr.rules); i++ {
		pr.expand(pr.rules[i])
	}

	// pr.index[g.base.first().rule] = 1
	// pr.rules = append(pr.rules, g.base.first().rule)

	log.Println("Len rules ", len(pr.rules))

	for i := 0; i < len(pr.rules); i++ {
		if _, err := fmt.Fprintf(w, "%d (%dx) (sz:%d)\t->", i, pr.rules[i].count, pr.rule_sizes[i]); err != nil {
			return err, 0, 0, 0
		}

		if err := pr.printExpanded(w, pr.rules[i]); err != nil {
			return err, 0, 0, 0
		}
	}

	return nil, pr.heads, pr.repeat, pr.indexes
}

// Parse parses the given bytes.
func Parse(str []byte) *Grammar {
	g := &Grammar{
		ruleID: maxRuneOrByte + 1,
		table:  make(digrams),
	}
	g.base = g.newRules()
	for off := 0; off < len(str); {
		var rb runeOrByte
		r, sz := utf8.DecodeRune(str[off:])
		if sz == 1 && r == utf8.RuneError {
			rb = newByte(str[off])
		} else {
			rb = newRune(r)
		}
		g.base.last().insertAfter(g.newSymbolFromValue(uint64(rb)))
		if off > 0 {
			g.base.last().prev.check()
		}
		off += sz
	}
	return g
}

// Parse parses the given bytes.
func ParseAddr(data []uint64) *Grammar {
	g := &Grammar{
		ruleID: maxInt64 + 1,
		table:  make(digrams),
	}
	g.base = g.newRules()
	for i := range data {

		g.base.last().insertAfter(g.newSymbolFromValue(data[i]))
		g.base.last().prev.check()
	}
	return g
}

// runeOrByte holds a rune or a byte so that we can distinguish between
// bytes that don't represent valid UTF-8 and all other runes. Values
// not representable as UTF-8 are in the range 128-255. All other
// runes are represented as 256 onwards (subtract 256 to get the
// actual rune value). Note that the range 0-127 is unused.
type runeOrByte rune

const maxRuneOrByte = uint64(utf8.MaxRune) + 256 // larger than the largest possible value of runeOrByte
const maxInt64 = 0xffff000000000000              // larger than the largest possible value of runeOrByte

func newRune(r rune) runeOrByte {
	return runeOrByte(r + 256)
}

// newByte returns a representation of the given byte b.
func newByte(b byte) runeOrByte {
	if b < utf8.RuneSelf {
		return runeOrByte(b) + 256
	}
	return runeOrByte(b)
}

// rune returns the rune representation of
// rb, or zero if there is none.
func (rb runeOrByte) rune() rune {
	if rb < 256 {
		return 0
	}
	return rune(rb - 256)
}

// appendEscaped appends the possibly escaped rune or byte
// to b. If it's printable, the printable representation is appended,
// otherwise \x, \u or \U are used as appropriate.
// Note, it doesn't escape \ itself.
func (rb runeOrByte) appendEscaped(b []byte) []byte {
	if rb < 256 {
		return append(b, fmt.Sprintf("\\x%02x", rb)...)
	}
	r := rune(rb - 256)
	switch {
	case unicode.IsPrint(r):
		return append(b, string(r)...)
	case r < utf8.RuneSelf:
		// Could use either representation, but \x is shorter.
		return append(b, fmt.Sprintf("\\x%02x", r)...)
	case r <= 0xffff:
		return append(b, fmt.Sprintf("\\u%04x", r)...)
	default:
		return append(b, fmt.Sprintf("\\U%08x", r)...)
	}
}

// appendBytes appends the byte (as a byte) or the rune (as utf-8)
// to b.
func (rb runeOrByte) appendBytes(b []byte) []byte {
	if rb < 256 {
		return append(b, byte(rb))
	}
	return append(b, string(rb-256)...)
}

// func ReadJson(data interface{}, filename string) {
// 	content, err := ioutil.ReadFile(filename)
// 	if err != nil {
// 		log.Fatalln(err)
// 	}

// 	err = json.Unmarshal(content, data)
// 	if err != nil {
// 		log.Fatalln("error:", err)
// 	}
// }
