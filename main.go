package main

import (
	"github.com/zlogic/nanorss-go/data"
)

func main() {
	service, _ := data.Open(data.DefaultOptions())
	defer service.Close()
}
