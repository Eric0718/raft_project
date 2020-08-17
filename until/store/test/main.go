package main

import (
	"kto/until/store/bg"
	"kto/until/store/server"
)

func main() {
	db := bg.New("test.db")
	s := server.New(6378, db)
	s.Run()
}
