package main

import (
	"flag"
	"nabil6339/opus-caf/caf"
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
