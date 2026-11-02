// Generate input caches with the unchanged dataset generator; no benchmarks run.
package main

import (
	"data-ownership-system/dataset"
	"flag"
	"log"
	"os"
)

func main() {
	n := flag.Int("n", 10000, "number of items")
	m := flag.Int("m", 10, "number of categories")
	size := flag.Int("size", 1024, "payload bytes per item")
	dir := flag.String("dir", "data", "cache directory")
	flag.Parse()
	if *n < 1 || *m < 1 || *size < 1 {
		log.Fatal("positive n, m and size required")
	}
	if err := os.MkdirAll(*dir, 0755); err != nil {
		log.Fatal(err)
	}
	cfg := &dataset.DatasetConfig{NumItems: *n, NumTypes: *m, ItemSize: *size, CompanyName: "TestCompany", Protocol: "https"}
	if _, _, err := dataset.NewDatasetGenerator(cfg, *dir).GenerateDataset(); err != nil {
		log.Fatal(err)
	}
}
