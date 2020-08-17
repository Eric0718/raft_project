package raftconsensus

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func readInfo() []byte {
	data, err := ioutil.ReadFile("./conf/clusterInfo.json")
	if err != nil {
		fmt.Print(err)
	}
	fmt.Println("==========", data)
	str := string(data)
	fmt.Println("=============", str)

	return data
}

func writeInfo(data []byte) {
	fp, err := os.OpenFile("./conf/clusterInfo.json", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()
	_, err = fp.Write(data)
	if err != nil {
		log.Fatal(err)
	}
}

// func main() {
// 	var data []byte
// 	data = testMarshal()
// 	fmt.Println(string(data))
// 	testWrite(data)
// 	data = testRead()
// 	testUnmarshal(data)
// }
