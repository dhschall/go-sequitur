package sequitur

// import (
// 	"encoding/json"
// 	"fmt"
// 	"io/ioutil"
// 	"log"
// )

// // func main() {

// // 	data := make(map[string][]uint64)
// // 	ReadJson(&data, "data.json")

// // 	fmt.Println("Length data", len(data))
// // }

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
