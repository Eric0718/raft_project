package main

import (
	"fmt"
	"kto/contract/exec"
	"kto/until/store/bg"
	"log"
)

func main() {
	db := bg.New("test.db")
	defer db.Close()

	// sc := parser.Parser([]byte("new \"USDT\" 1000000000"))
	// e, err := exec.New(db, sc, "wxh")
	// if err != nil {
	// 	fmt.Println("===", err)
	// 	return
	// }
	// fmt.Printf("%x\n", e.Root())
	// e.Flush()

	// sc1 := parser.Parser([]byte("mint \"USDT\" 1000000000"))
	// e, err := exec.New(db, sc1, "wxh")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("%x\n", e.Root())
	// e.Flush()

	// "transfer \"abc\" 10 \"to\""

	// sc1 := parser.Parser([]byte("transfer \"USDT\" 5 \"zyh\""))
	// e1, err := exec.New(db, sc1, "lzl")
	// if err != nil {
	// 	fmt.Println("===", err)
	// 	return
	// }
	// fmt.Printf("%x\n", e1.Root())
	// e1.Flush()

	b, err := exec.Balance(db, "USDT", "wxh") // 2，代比， 3，地址
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v\n", b)
}
