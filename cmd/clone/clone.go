package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/HardDie/harddie_go_template/internal/clone"
	"github.com/HardDie/harddie_go_template/internal/logger"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: applicator create $appname $dir\n")
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	c := clone.New()

	if err := c.Run(args); err != nil {
		if errors.Is(err, clone.ErrInvalidParams) {
			usage()
		} else {
			logger.Error(err.Error())
			panic(err)
		}
	}
}
