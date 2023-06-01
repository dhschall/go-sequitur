package sequitur

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
)

func main() {

	data := make(map[string][]uint64)
	ReadJson(&data, "data.json")

	fmt.Println("Length data", len(data))

	n := 18

	d1 := data[fmt.Sprintf("%d", n)]

	fmt.Println("Length d1 ", len(d1))
	d2 := d1[:]

	// fmt.Println("Length d2: ", d2)
	for i := range d2 {
		if d2[i] > math.MaxInt64 {
			d2[i] -= (1 << 63)
		}
	}
	// fmt.Println("Length d2: ", d2)

	g := ParseAddr(d2)

	fmt.Println("Length r: ", len(g.table))

	var output bytes.Buffer
	// if err := g.Print(&output); err != nil {
	// 	panic(err)
	// }
	if err := g.PrettyPrint(&output); err != nil {
		panic(err)
	}

	// fmt.Println(string(output.Bytes()))

	filename := fmt.Sprintf("seq_grammar_%d.txt", n)

	ioutil.WriteFile(filename, output.Bytes(), 0644)

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
