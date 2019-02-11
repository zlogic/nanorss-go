package main

import (
	"github.com/zlogic/nanorss/data"
)

func main() {
	service, _ := data.Open(data.DefaultOptions())
	defer service.Close()
}
