package main

import (
	"data-ownership-system/preprocessor"
	"flag"
	"fmt"
	"os"
)

// C:\Users\dell\Desktop\CLEVR_v1.0

func main() {
	var (
		dataPath    = flag.String("path", "", "real data directory path (required)")
		protocol    = flag.String("protocol", "http", "URL protocol")
		companyName = flag.String("company", "openslr.org", "company name")
		outputFile  = flag.String("output", "LibriSpeech.json", "output JSON filename")
	)
	flag.Parse()

	if *dataPath == "" {
		fmt.Println("error: data path is required")
		fmt.Println("Usage:")
		flag.Usage()
		os.Exit(1)
	}

	err := preprocessor.PreprocessRealData(*dataPath, *protocol, *companyName, *outputFile)
	if err != nil {
		fmt.Printf("preprocessing failed: %v\n", err)
		os.Exit(1)
	}
}
