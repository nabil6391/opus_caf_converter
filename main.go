package main

import (
	"flag"

	"github.com/nabil6391/opus_caf_converter/caf"
)

func main() {
	inputFile := ""
	outputFile := ""

	flag.StringVar(&inputFile, "i", "", "input file")
	flag.StringVar(&outputFile, "o", "", "output file")

	flag.Parse()

	if inputFile == "" || outputFile == "" {
		flag.Usage()
		return
	}

	if err := caf.ConvertOpusToCaf(inputFile, outputFile); err != nil {
		panic(err)
	}
}
