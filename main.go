package main

import (
	"bytes"
	"crypto/rand"
	"data-ownership-system/auth"
	"data-ownership-system/dataset"
	"data-ownership-system/pkg/types"
	"data-ownership-system/utils"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	dataItemSize = 1024
	numDataItems = 10000
	numDataTypes = 10

	loadFactor = 0.75

	testIterations = 1000

	companyName = "TestCompany"
	protocol    = "https"

	useRealDataset  = false
	realDatasetPath = "LibriSpeech.json"
)

func main() {
	emitProof := flag.Bool("emit-proof", false, "emit one layered proof as JSON and exit")
	emitIndex := flag.Int("emit-index", 0, "data item index used for proof generation (default 0)")
	repeatTimes := flag.Int("repeat", 10, "number of benchmark runs (default 10)")
	silent := flag.Bool("silent", false, "suppress benchmark progress output")
	flag.IntVar(&numDataItems, "n", numDataItems, "number of synthetic data items")
	flag.IntVar(&numDataTypes, "m", numDataTypes, "number of synthetic data types")
	flag.IntVar(&dataItemSize, "size", dataItemSize, "synthetic data item size in bytes")
	flag.IntVar(&testIterations, "iterations", testIterations, "per-operation benchmark iterations")
	flag.Float64Var(&loadFactor, "load", loadFactor, "hash table load factor")
	flag.BoolVar(&useRealDataset, "real", useRealDataset, "load a preprocessed real dataset from data/")
	flag.StringVar(&realDatasetPath, "dataset", realDatasetPath, "preprocessed real dataset JSON filename")
	flag.StringVar(&companyName, "company", companyName, "company name used in generated URLs")
	flag.StringVar(&protocol, "protocol", protocol, "URL protocol used in generated URLs")
	flag.Parse()

	if *silent && !*emitProof {
		devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err == nil {
			os.Stdout = devNull
		}
	}

	fmt.Println(strings.Repeat("=", 50))

	if err := validateConfiguration(); err != nil {
		log.Fatalf("configuration validation failed: %v", err)
	}

	printConfiguration()

	if *emitProof {
		runSingleTest(*emitIndex)
		return
	}

	if *repeatTimes > 1 {
		runMultipleTests(*repeatTimes)
	} else {
		runSingleTest(-1)
	}
}

func runSingleTest(emitIndex int) {
	perfSuite := utils.NewPerformanceSuite()

	fmt.Println("\nStep 1: generate/load dataset")
	dataItems, dataTypes, dataGenerator, err := generateDataset()
	if err != nil {
		log.Fatalf("dataset generation failed: %v", err)
	}

	fmt.Println("\nStep 2: build authenticated data structure")
	fmt.Printf("parallelism: %d data types = %d concurrent goroutines\n", numDataTypes, numDataTypes)

	ads, buildTime := buildAuthDataStructure(dataItems, dataTypes, dataGenerator)
	perfSuite.AddResult("Build authenticated data structure", buildTime)
	printBuildResults(ads, buildTime)

	fmt.Println("\nStep 3: proof path generation test")
	proofGenTime := testProofGeneration(ads, dataItems, perfSuite)
	fmt.Printf("Average proof path generation time: %.0f ns (%.3f ms)\n", proofGenTime, proofGenTime/1e6)

	if emitIndex >= 0 {
		if err := emitOneProofJSON(ads, dataItems, emitIndex); err != nil {
			log.Fatalf("failed to emit proof: %v", err)
		}
		return
	}

	fmt.Println("\nStep 3.5: proof path walkthrough and verification simulation")
	demonstrateProofProcess(ads, dataItems)

	fmt.Println("\nStep 4: dynamic operation tests")
	testDynamicOperations(ads, dataItems, dataTypes, perfSuite)

	perfSuite.PrintAllResults()

	printSystemStatistics(ads)

	fmt.Println("\nSystem test completed!")
}

func runMultipleTests(times int) {
	fmt.Printf("\n========== Start %d benchmark runs ==========\n", times)

	var buildTimes []float64
	var proofGenTimes []float64
	var addTimes []float64
	var updateTimes []float64
	var deleteTimes []float64

	fmt.Println("\nPreparation: generate/load dataset")
	dataItems, dataTypes, dataGenerator, err := generateDataset()
	if err != nil {
		log.Fatalf("dataset generation failed: %v", err)
	}
	fmt.Printf("Dataset ready: %d data items, %d data types\n", len(dataItems), len(dataTypes))

	for i := 1; i <= times; i++ {
		fmt.Printf("\n========== Run %d/%d ==========\n", i, times)

		fmt.Println("Test: build authenticated data structure")
		ads, buildTime := buildAuthDataStructure(dataItems, dataTypes, dataGenerator)
		buildTimes = append(buildTimes, buildTime)
		fmt.Printf("  Build time: %.3f ms\n", buildTime/1e6)

		fmt.Println("Test: proof generation")
		proofGenTime := testProofGenerationSimple(ads, dataItems)
		proofGenTimes = append(proofGenTimes, proofGenTime)
		fmt.Printf("  Proof generation time: %.3f ms\n", proofGenTime/1e6)

		fmt.Println("Test: dynamic operations")
		addTime, updateTime, deleteTime := testDynamicOperationsSimple(ads, dataItems, dataTypes, dataGenerator)
		addTimes = append(addTimes, addTime)
		updateTimes = append(updateTimes, updateTime)
		deleteTimes = append(deleteTimes, deleteTime)
		fmt.Printf("  Add operation time: %.3f ms\n", addTime/1e6)
		fmt.Printf("  Update operation time: %.3f ms\n", updateTime/1e6)
		fmt.Printf("  Delete operation time: %.3f ms\n", deleteTime/1e6)
	}

	fmt.Printf("\n========== %d benchmark average results ==========\n", times)
	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Number of data items (n): %d\n", numDataItems)
	fmt.Printf("  Number of data types (m): %d\n", numDataTypes)
	fmt.Printf("  Item size: %d bytes (%.1f KB)\n", dataItemSize, float64(dataItemSize)/1024)
	fmt.Printf("  Test iterations: %d\n", testIterations)

	avgBuildTime := average(buildTimes)
	avgProofGenTime := average(proofGenTimes)
	avgAddTime := average(addTimes)
	avgUpdateTime := average(updateTimes)
	avgDeleteTime := average(deleteTimes)

	fmt.Printf("\nBenchmark results (average):\n")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-25s | %15s | %15s\n", "Operation", "Average time (ns)", "Average time (ms)")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-25s | %15.0f | %15.3f\n", "Build authenticated data structure", avgBuildTime, avgBuildTime/1e6)
	fmt.Printf("%-25s | %15.0f | %15.3f\n", "Proof generation", avgProofGenTime, avgProofGenTime/1e6)
	fmt.Printf("%-25s | %15.0f | %15.3f\n", "Add operation", avgAddTime, avgAddTime/1e6)
	fmt.Printf("%-25s | %15.0f | %15.3f\n", "Update operation", avgUpdateTime, avgUpdateTime/1e6)
	fmt.Printf("%-25s | %15.0f | %15.3f\n", "Delete operation", avgDeleteTime, avgDeleteTime/1e6)
	fmt.Println(strings.Repeat("-", 60))

	fmt.Printf("\nStandard deviation:\n")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-25s | %15s | %15s\n", "Operation", "Std. dev. (ns)", "Std. dev. (ms)")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-25s | %15.0f | %15.3f\n", "Build authenticated data structure", stdDev(buildTimes, avgBuildTime), stdDev(buildTimes, avgBuildTime)/1e6)
	fmt.Printf("%-25s | %15.0f | %15.3f\n", "Proof generation", stdDev(proofGenTimes, avgProofGenTime), stdDev(proofGenTimes, avgProofGenTime)/1e6)
	fmt.Printf("%-25s | %15.0f | %15.3f\n", "Add operation", stdDev(addTimes, avgAddTime), stdDev(addTimes, avgAddTime)/1e6)
	fmt.Printf("%-25s | %15.0f | %15.3f\n", "Update operation", stdDev(updateTimes, avgUpdateTime), stdDev(updateTimes, avgUpdateTime)/1e6)
	fmt.Printf("%-25s | %15.0f | %15.3f\n", "Delete operation", stdDev(deleteTimes, avgDeleteTime), stdDev(deleteTimes, avgDeleteTime)/1e6)
	fmt.Println(strings.Repeat("-", 60))

	fmt.Printf("\n========== Benchmark completed! ==========\n")
}

func validateConfiguration() error {
	if dataItemSize <= 0 {
		return fmt.Errorf("item size must be greater than 0, current value: %d", dataItemSize)
	}
	if dataItemSize > 10*1024*1024 {
		return fmt.Errorf("item size is too large; maximum is 10MB, current value: %d bytes", dataItemSize)
	}

	if numDataItems <= 0 {
		return fmt.Errorf("number of data items must be greater than 0, current value: %d", numDataItems)
	}
	if numDataItems > 10000000 {
		return fmt.Errorf("too many data items; maximum is 10,000,000, current value: %d", numDataItems)
	}

	if numDataTypes <= 0 {
		return fmt.Errorf("number of data types must be greater than 0, current value: %d", numDataTypes)
	}
	if numDataTypes > 5000 {
		return fmt.Errorf("too many data types; maximum is 5000, current value: %d", numDataTypes)
	}

	if loadFactor <= 0 || loadFactor >= 1.0 {
		return fmt.Errorf("load factor must be in (0,1), current value: %.2f", loadFactor)
	}

	if testIterations <= 0 {
		return fmt.Errorf("test iterations must be greater than 0, current value: %d", testIterations)
	}
	if testIterations > 1000 {
		return fmt.Errorf("too many test iterations; maximum is 1000, current value: %d", testIterations)
	}

	if companyName == "" {
		return fmt.Errorf("company name cannot be empty")
	}

	if protocol == "" {
		return fmt.Errorf("protocol cannot be empty")
	}

	if numDataItems > 5000000 {
		return fmt.Errorf("too many data items; maximum is 5,000,000, current value %d", numDataItems)
	}

	if dataItemSize > 100*1024*1024 {
		return fmt.Errorf("item size is too large; maximum is 100MB, current value %.2fMB",
			float64(dataItemSize)/(1024*1024))
	}

	return nil
}

func printConfiguration() {
	fmt.Println("Configuration:")
	if useRealDataset {
		fmt.Printf("  Dataset mode: real dataset\n")
		fmt.Printf("  Dataset file: %s\n", realDatasetPath)
		fmt.Printf("  Number of data items (n): %d (loaded from real dataset)\n", numDataItems)
		fmt.Printf("  Number of data types (m): %d (loaded from real dataset)\n", numDataTypes)
		fmt.Printf("  Average item size: %d bytes (%.1f KB)\n", dataItemSize, float64(dataItemSize)/1024)
	} else {
		fmt.Printf("  Dataset mode: synthetic dataset\n")
		fmt.Printf("  Number of data items (n): %d\n", numDataItems)
		fmt.Printf("  Number of data types (m): %d\n", numDataTypes)
		fmt.Printf("  Item size: %d bytes (%.1f KB)\n", dataItemSize, float64(dataItemSize)/1024)
	}
	fmt.Printf("  Load factor: %.2f\n", loadFactor)
	fmt.Printf("  parallelism: auto by number of data types\n")
	fmt.Printf("  Test iterations: %d\n", testIterations)
	fmt.Printf("  Company: %s\n", companyName)
	fmt.Printf("  Protocol: %s\n", protocol)
}

func generateDataset() ([]*types.DataItem, []string, *dataset.DatasetGenerator, error) {
	dataDir := "data"
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	if useRealDataset {
		fmt.Printf("Using real dataset: %s\n", realDatasetPath)
		return loadRealDataset(dataDir)
	} else {
		fmt.Printf("Using synthetic dataset: n=%d, m=%d\n", numDataItems, numDataTypes)
		return generateSimulatedDataset(dataDir)
	}
}

func generateSimulatedDataset(dataDir string) ([]*types.DataItem, []string, *dataset.DatasetGenerator, error) {
	config := &dataset.DatasetConfig{
		NumItems:    numDataItems,
		NumTypes:    numDataTypes,
		ItemSize:    dataItemSize,
		CompanyName: companyName,
		Protocol:    protocol,
	}

	generator := dataset.NewDatasetGenerator(config, dataDir)

	dataItems, dataTypes, err := generator.GenerateDataset()

	return dataItems, dataTypes, generator, err
}

func loadRealDataset(dataDir string) ([]*types.DataItem, []string, *dataset.DatasetGenerator, error) {
	realDatasetFullPath := filepath.Join(dataDir, realDatasetPath)

	if _, err := os.Stat(realDatasetFullPath); os.IsNotExist(err) {
		return nil, nil, nil, fmt.Errorf("real dataset file does not exist: %s\nplease generate the dataset with the preprocessor first", realDatasetFullPath)
	}

	file, err := os.Open(realDatasetFullPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open real dataset file: %v", err)
	}
	defer file.Close()

	var realDataset struct {
		Config    dataset.DatasetConfig `json:"config"`
		DataTypes []string              `json:"data_types"`
		Items     []struct {
			URL     string `json:"url"`
			Content string `json:"content"`
			Hash    string `json:"hash"`
		} `json:"items"`
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&realDataset)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse real dataset JSON: %v", err)
	}

	dataItems := make([]*types.DataItem, len(realDataset.Items))
	typeCounts := make(map[string]int)

	for _, dataType := range realDataset.DataTypes {
		typeCounts[dataType] = 0
	}

	for i, item := range realDataset.Items {
		hashBytes, err := hex.DecodeString(item.Hash)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse hash %s: %v", item.URL, err)
		}

		dataItems[i] = &types.DataItem{
			URL:     item.URL,
			Content: []byte{},
			Hash:    hashBytes,
		}

		dataType := extractDataTypeFromURL(item.URL)
		if _, exists := typeCounts[dataType]; exists {
			typeCounts[dataType]++
		}
	}

	generator := createRealDatasetGenerator(realDataset.Config, realDataset.DataTypes, typeCounts, dataDir)

	numDataItems = realDataset.Config.NumItems
	numDataTypes = realDataset.Config.NumTypes
	dataItemSize = realDataset.Config.ItemSize
	companyName = realDataset.Config.CompanyName
	protocol = realDataset.Config.Protocol

	fmt.Printf("Loaded real dataset: %d data items, %d data types\n", len(dataItems), len(realDataset.DataTypes))
	fmt.Println("Real dataset type distribution:")
	for _, dataType := range realDataset.DataTypes {
		count := typeCounts[dataType]
		percentage := float64(count) / float64(len(dataItems)) * 100
		fmt.Printf("  %s: %ditems (%.1f%%)\n", dataType, count, percentage)
	}

	return dataItems, realDataset.DataTypes, generator, nil
}

func createRealDatasetGenerator(config dataset.DatasetConfig, dataTypes []string, typeCounts map[string]int, dataDir string) *dataset.DatasetGenerator {
	generator := dataset.NewDatasetGenerator(&config, dataDir)

	return generator
}

func buildAuthDataStructure(dataItems []*types.DataItem, dataTypes []string, dataGenerator *dataset.DatasetGenerator) (*auth.AuthenticatedDataStructure, float64) {
	timer := utils.NewTimer("Build authenticated data structure")
	timer.Start()

	ads := auth.NewAuthenticatedDataStructure(loadFactor)

	typeCounts := dataGenerator.GetLastTypeCounts()
	err := ads.BuildFromDataItems(dataItems, dataTypes, typeCounts)
	if err != nil {
		log.Fatalf("failed to build authenticated data structure: %v", err)
	}

	buildTime := timer.Stop()

	return ads, buildTime
}

func printBuildResults(ads *auth.AuthenticatedDataStructure, buildTime float64) {
	fmt.Printf("Build completed, elapsed: %.0f ns (%.3f ms)\n", buildTime, buildTime/1e6)
	fmt.Printf("Top root hash: %x\n", ads.GetTopRoot()[:16])

	stats := ads.GetStatistics()
	fmt.Printf("Authenticated data structure statistics:\n")
	fmt.Printf("  Number of data types: %d\n", stats["data_types"])
	fmt.Printf("  Total data items: %d\n", stats["total_items"])
	fmt.Printf("  Total hash table capacity: %d\n", stats["total_capacity"])

	if useRealDataset {
		fmt.Printf("Real dataset per-type details:\n")
	} else {
		fmt.Printf("Synthetic dataset per-type details:\n")
	}

	for _, typeName := range ads.DataTypes {
		if countKey := fmt.Sprintf("%s_count", typeName); stats[countKey] != nil {
			count := stats[countKey]
			capacity := stats[fmt.Sprintf("%s_capacity", typeName)]
			loadFactor := stats[fmt.Sprintf("%s_load_factor", typeName)]

			treeHeight := calculateMerkleTreeHeight(capacity.(int))

			fmt.Printf("  %s: count=%v, capacity=%v, load=%.2f, tree height=%d\n",
				typeName, count, capacity, loadFactor, treeHeight)
		}
	}
}

func calculateMerkleTreeHeight(capacity int) int {
	if capacity <= 1 {
		return 0
	}
	height := 0
	nodes := capacity
	for nodes > 1 {
		nodes = (nodes + 1) / 2
		height++
	}
	return height
}

func testProofGeneration(ads *auth.AuthenticatedDataStructure, dataItems []*types.DataItem, perfSuite *utils.PerformanceSuite) float64 {
	timer := utils.NewTimer("Proof path generation")
	timer.Start()
	validCount := 0
	validProofs := make(map[int]*auth.CombinedProof)
	for i := 0; i < testIterations; i++ {
		index := i % len(dataItems)
		item := dataItems[index]
		proof, err := ads.GenerateProof(item.URL, item.Hash)

		if err == nil && proof != nil {
			validCount++
			validProofs[i] = proof
		}
	}
	genDuration := timer.Stop()
	avgGenTime := genDuration / float64(testIterations)
	perfSuite.AddResult("Proof path generation", avgGenTime)

	timer2 := utils.NewTimer("Proof path verification")
	timer2.Start()
	verifyValidCount := 0
	for i := 0; i < testIterations; i++ {
		if proof, exists := validProofs[i]; exists {
			isValid := ads.VerifyProof(proof)
			if isValid {
				verifyValidCount++
			}
		}
	}
	verifyDuration := timer2.Stop()
	avgVerifyTime := verifyDuration / float64(testIterations)
	perfSuite.AddResult("Proof path verification", avgVerifyTime)

	fmt.Printf("Proof generation: avg %.1f ns (%.3f ms), valid %d/%d\n", avgGenTime, avgGenTime/1e6, validCount, testIterations)
	fmt.Printf("Proof verification: avg %.1f ns (%.3f ms), valid %d/%d\n", avgVerifyTime, avgVerifyTime/1e6, verifyValidCount, testIterations)

	return avgGenTime
}

func testDynamicOperations(ads *auth.AuthenticatedDataStructure, dataItems []*types.DataItem, dataTypes []string, perfSuite *utils.PerformanceSuite) {
	config := &dataset.DatasetConfig{
		NumItems:    numDataItems,
		NumTypes:    numDataTypes,
		ItemSize:    dataItemSize,
		CompanyName: companyName,
		Protocol:    protocol,
	}
	generator := dataset.NewDatasetGenerator(config, "data")

	fmt.Println("\nTest add operation:")
	testAddOperation(ads, generator, dataTypes, perfSuite)

	fmt.Println("\nTest update operation:")
	testUpdateOperation(ads, dataItems, perfSuite)

	fmt.Println("\nTest delete operation:")
	testDeleteOperation(ads, dataItems, perfSuite)
}

func testAddOperation(ads *auth.AuthenticatedDataStructure, generator *dataset.DatasetGenerator, dataTypes []string, perfSuite *utils.PerformanceSuite) {

	beforeCounts := make(map[string]int)
	for dt, ht := range ads.HashTables {
		beforeCounts[dt] = ht.Count()
	}
	beforeRoot := ads.GetTopRoot()

	items := make([]*types.DataItem, testIterations)
	for i := 0; i < testIterations; i++ {
		newItem := generator.GenerateRandomDataItem(dataTypes)
		newItem.URL = fmt.Sprintf("%s_test_add_%d_%d", newItem.URL, time.Now().UnixNano(), i)
		items[i] = newItem
	}

	timer := utils.NewTimer("Add operation")
	successCount := 0
	rootChanges := 0
	var totalDuration float64 = 0

	for i := 0; i < testIterations; i++ {
		prevRoot := ads.GetTopRoot()

		timer.Start()
		err := ads.InsertDataItem(items[i])
		opDuration := timer.Stop()
		totalDuration += opDuration

		newRoot := ads.GetTopRoot()

		if err == nil {
			successCount++
			if !bytes.Equal(prevRoot, newRoot) {
				rootChanges++
			}

			dt := extractDataTypeFromURL(items[i].URL)
			if ht := ads.HashTables[dt]; ht != nil {
				if _, found := ht.Get(items[i].URL); !found {
					log.Printf("add validation failed: hash table does not contain %s", items[i].URL)
				}
				if proof, err := ads.GenerateProof(items[i].URL, items[i].Hash); err != nil || !ads.VerifyProof(proof) {
					log.Printf("add validation failed: could not generate a valid proof for %s", items[i].URL)
				}
			}
		}
	}
	avgDuration := totalDuration / float64(testIterations)
	perfSuite.AddResult("Add operation", avgDuration)

	afterRoot := ads.GetTopRoot()
	totalCountIncrease := 0
	for dt, ht := range ads.HashTables {
		totalCountIncrease += ht.Count() - beforeCounts[dt]
	}

	fmt.Printf("Add operation: avg %.1f ns (%.3f ms), success %d/%d, root changes %d, count delta %d\n",
		avgDuration, avgDuration/1e6, successCount, testIterations, rootChanges, totalCountIncrease)
	fmt.Printf("  root hash: %x -> %x\n", beforeRoot[:8], afterRoot[:8])
}

func testUpdateOperation(ads *auth.AuthenticatedDataStructure, dataItems []*types.DataItem, perfSuite *utils.PerformanceSuite) {
	startIndex := len(dataItems) / 2

	beforeRoot := ads.GetTopRoot()

	timer := utils.NewTimer("Update operation")
	successCount := 0
	rootChanges := 0
	var totalDuration float64 = 0

	for i := 0; i < testIterations; i++ {
		idx := startIndex + (i % (len(dataItems) - startIndex))
		item := dataItems[idx]

		dt := extractDataTypeFromURL(item.URL)
		var oldHash []byte
		if ht := ads.HashTables[dt]; ht != nil {
			if entry, ok := ht.Get(item.URL); ok {
				oldHash = append([]byte(nil), entry.Hash...)
			}
		}

		newContent := make([]byte, dataItemSize)
		rand.Read(newContent)
		newHash := utils.Keccak256(newContent)

		prevRoot := ads.GetTopRoot()

		timer.Start()
		err := ads.UpdateDataItem(item.URL, newHash)
		opDuration := timer.Stop()
		totalDuration += opDuration

		newRoot := ads.GetTopRoot()

		if err == nil {
			successCount++
			if !bytes.Equal(prevRoot, newRoot) {
				rootChanges++
			}

			if proof, err := ads.GenerateProof(item.URL, newHash); err != nil || !ads.VerifyProof(proof) {
				log.Printf("update validation failed: new hash cannot generate a valid proof for %s", item.URL)
			}
			if oldHash != nil {
				if _, err := ads.GenerateProof(item.URL, oldHash); err == nil {
					log.Printf("update validation failed: old hash still generates a proof for %s", item.URL)
				}
			}
		}
	}
	avgDuration := totalDuration / float64(testIterations)
	perfSuite.AddResult("Update operation", avgDuration)

	afterRoot := ads.GetTopRoot()
	fmt.Printf("Update operation: avg %.1f ns (%.3f ms), success %d/%d, root changes %d\n",
		avgDuration, avgDuration/1e6, successCount, testIterations, rootChanges)
	fmt.Printf("  root hash: %x -> %x\n", beforeRoot[:8], afterRoot[:8])
}

func testDeleteOperation(ads *auth.AuthenticatedDataStructure, dataItems []*types.DataItem, perfSuite *utils.PerformanceSuite) {
	maxIndex := len(dataItems) / 2

	beforeCounts := make(map[string]int)
	for dt, ht := range ads.HashTables {
		beforeCounts[dt] = ht.Count()
	}
	beforeRoot := ads.GetTopRoot()

	timer := utils.NewTimer("Delete operation")
	successCount := 0
	rootChanges := 0
	var totalDuration float64 = 0
	actualOperations := 0

	for i := 0; i < testIterations && i < maxIndex; i++ {
		item := dataItems[i]

		prevRoot := ads.GetTopRoot()

		timer.Start()
		err := ads.DeleteDataItem(item.URL)
		opDuration := timer.Stop()
		totalDuration += opDuration
		actualOperations++

		newRoot := ads.GetTopRoot()

		if err == nil {
			successCount++
			if !bytes.Equal(prevRoot, newRoot) {
				rootChanges++
			}

			dt := extractDataTypeFromURL(item.URL)
			if ht := ads.HashTables[dt]; ht != nil {
				if _, found := ht.Get(item.URL); found {
					log.Printf("delete validation failed: hash table still contains %s", item.URL)
				}
			}
			if proof, err := ads.GenerateProof(item.URL, item.Hash); err == nil && proof != nil {
				log.Printf("delete validation failed: proof still generated for %s", item.URL)
			}
		}
	}
	avgDuration := totalDuration / float64(actualOperations)
	perfSuite.AddResult("Delete operation", avgDuration)

	afterRoot := ads.GetTopRoot()
	totalCountDecrease := 0
	for dt, ht := range ads.HashTables {
		totalCountDecrease += beforeCounts[dt] - ht.Count()
	}

	fmt.Printf("Delete operation: avg %.1f ns (%.3f ms), success %d/%d, root changes %d, count delta %d\n",
		avgDuration, avgDuration/1e6, successCount, actualOperations, rootChanges, totalCountDecrease)
	fmt.Printf("  root hash: %x -> %x\n", beforeRoot[:8], afterRoot[:8])
}

func printSystemStatistics(ads *auth.AuthenticatedDataStructure) {
	fmt.Println("\nSystem statistics:")

	stats := ads.GetStatistics()
	fmt.Printf("Final state:\n")
	fmt.Printf("  Dataset mode: %s\n", getDatasetModeString())
	fmt.Printf("  Number of data types: %d\n", stats["data_types"])
	fmt.Printf("  Total data items: %d\n", stats["total_items"])
	fmt.Printf("  Total hash table capacity: %d\n", stats["total_capacity"])
	fmt.Printf("  Final root hash: %s\n", stats["top_root"])

	totalCapacity := stats["total_capacity"].(int)

	hashTableMemory := totalCapacity * (256 + 64 + 8)

	merkleTreeMemory := totalCapacity * 2 * 50

	totalEstimatedMemory := hashTableMemory + merkleTreeMemory

	originalDataSize := int64(numDataItems) * int64(dataItemSize)
	overheadPercentage := float64(totalEstimatedMemory) / float64(originalDataSize) * 100

	fmt.Printf("  Estimated memory use: %.2f KB (%.2f MB)\n",
		float64(totalEstimatedMemory)/1024, float64(totalEstimatedMemory)/(1024*1024))
	fmt.Printf("    - Hash table memory: %.2f MB\n", float64(hashTableMemory)/(1024*1024))
	fmt.Printf("    - Merkle tree memory: %.2f MB\n", float64(merkleTreeMemory)/(1024*1024))

	if useRealDataset {
		fmt.Printf("  Real dataset size: %.2f MB (estimated from average file size)\n", float64(originalDataSize)/(1024*1024))
		fmt.Printf("  Authentication structure storage overhead: %.2f MB (%.1f%% of estimated data size)\n",
			float64(totalEstimatedMemory)/(1024*1024), overheadPercentage)
	} else {
		fmt.Printf("  Synthetic dataset size: %.2f MB\n", float64(originalDataSize)/(1024*1024))
		fmt.Printf("  Authentication structure storage overhead: %.2f MB (%.1f%% of raw data)\n",
			float64(totalEstimatedMemory)/(1024*1024), overheadPercentage)
	}

	dataDir := "data"
	if files, err := filepath.Glob(filepath.Join(dataDir, "*.json")); err == nil {
		fmt.Printf("  Number of dataset files: %d\n", len(files))
		if useRealDataset {
			realDatasetFullPath := filepath.Join(dataDir, realDatasetPath)
			if info, err := os.Stat(realDatasetFullPath); err == nil {
				fmt.Printf("  Dataset file used: %s (%.2f KB)\n", realDatasetPath, float64(info.Size())/1024)
			}
		} else {
			if len(files) > 0 {
				if info, err := os.Stat(files[0]); err == nil {
					fmt.Printf("  Most recent dataset file size: %.2f KB\n", float64(info.Size())/1024)
				}
			}
		}
	}
}

func getDatasetModeString() string {
	if useRealDataset {
		return "real dataset"
	}
	return "synthetic dataset"
}

func demonstrateProofProcess(ads *auth.AuthenticatedDataStructure, dataItems []*types.DataItem) {
	index := 0
	if len(dataItems) > 0 {
		if len(dataItems) > 43 {
			index = 42
		} else {
			index = 0
		}
	}
	demoItem := dataItems[index]

	fmt.Printf("=== Proof path walkthrough ===\n")
	fmt.Printf("Demo data item:\n")
	fmt.Printf("  URL: %s\n", demoItem.URL)
	fmt.Printf("  Hash: %x\n", demoItem.Hash)

	fmt.Printf("\n1. Generate proof path:\n")
	proof, err := ads.GenerateProof(demoItem.URL, demoItem.Hash)
	if err != nil {
		fmt.Printf("proof generation failed: %v\n", err)
		return
	}

	fmt.Printf("  Data type: %s\n", proof.DataType)
	fmt.Printf("  Merkle proof path:\n")
	fmt.Printf("    Leaf index: %d\n", proof.MerkleProof.Index)
	fmt.Printf("    Sibling path (%d levels):\n", len(proof.MerkleProof.Siblings))

	for i, sibling := range proof.MerkleProof.Siblings {
		fmt.Printf("      Level %d sibling: %x\n", i+1, sibling[:8])
	}

	fmt.Printf("    Subtree root hash: %x\n", proof.MerkleProof.Root[:8])

	fmt.Printf("\n  Other subtree roots:\n")
	for _, dataType := range ads.DataTypes {
		if dataType != proof.DataType {
			if root, exists := proof.OtherRoots[dataType]; exists {
				fmt.Printf("    %s: %x\n", dataType, root[:8])
			}
		}
	}

	fmt.Printf("  Top root hash: %x\n", proof.TopRoot[:8])

	fmt.Printf("\n2. Verification simulation:\n")

	fmt.Printf("  Step 1: verify Merkle proof path\n")
	fmt.Printf("    Compute upward from the leaf...\n")

	entryData := []byte(proof.URL)
	entryData = append(entryData, byte(0)) // state = 0
	entryData = append(entryData, proof.Hash...)
	currentHash := utils.Keccak256(entryData)

	fmt.Printf("    Leaf hash: %x\n", currentHash[:8])

	currentIndex := proof.MerkleProof.Index
	for i, siblingHash := range proof.MerkleProof.Siblings {
		if currentIndex%2 == 0 {
			combinedHash := append(currentHash, siblingHash...)
			currentHash = utils.Keccak256(combinedHash)
			fmt.Printf("    Level %d: left child, combine with sibling %x -> %x\n",
				i+1, siblingHash[:8], currentHash[:8])
		} else {
			combinedHash := append(siblingHash, currentHash...)
			currentHash = utils.Keccak256(combinedHash)
			fmt.Printf("    Level %d: right child, combine with sibling %x -> %x\n",
				i+1, siblingHash[:8], currentHash[:8])
		}
		currentIndex = currentIndex / 2
	}

	merkleValid := bytes.Equal(currentHash, proof.MerkleProof.Root)
	fmt.Printf("    Merkle verification result: %s (computed root: %x, expected root: %x)\n",
		getValidationResult(merkleValid), currentHash[:8], proof.MerkleProof.Root[:8])

	fmt.Printf("  Step 2: verify top root hash\n")
	fmt.Printf("    Recompute the top root in data-type order...\n")

	var combinedHash []byte
	fmt.Printf("    Concatenation order: ")
	for i, dataType := range ads.DataTypes {
		if i > 0 {
			fmt.Printf(" + ")
		}
		fmt.Printf("%s", dataType)

		if dataType == proof.DataType {
			combinedHash = append(combinedHash, proof.MerkleProof.Root...)
		} else {
			if root, exists := proof.OtherRoots[dataType]; exists {
				combinedHash = append(combinedHash, root...)
			}
		}
	}
	fmt.Printf("\n")

	calculatedTopRoot := utils.Keccak256(combinedHash)
	topRootValid := bytes.Equal(calculatedTopRoot, proof.TopRoot)

	fmt.Printf("    Computed top root: %x\n", calculatedTopRoot[:8])
	fmt.Printf("    Expected top root: %x\n", proof.TopRoot[:8])
	fmt.Printf("    Top root verification result: %s\n", getValidationResult(topRootValid))

	finalResult := merkleValid && topRootValid
	fmt.Printf("\n3. Final verification result: %s\n", getValidationResult(finalResult))

	if finalResult {
		fmt.Printf("   [OK] Data item belongs to this data type Merkle tree\n")
		fmt.Printf("   [OK] Top root verification passed; data was not tampered with\n")
		fmt.Printf("   [OK] Ownership verification succeeded and can support arbitration\n")
	} else {
		fmt.Printf("   [FAIL] Verification failed; data may be tampered with or proof is invalid\n")
	}

	fmt.Printf("\n=== Proof path walkthrough completed ===\n")
	fmt.Println("\nSystem test completed!")
}

func getValidationResult(valid bool) string {
	if valid {
		return "PASS"
	}
	return "FAIL"
}

func extractDataTypeFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return "default"
}

func testProofGenerationSimple(ads *auth.AuthenticatedDataStructure, dataItems []*types.DataItem) float64 {
	timer := utils.NewTimer("Proof path generation")
	timer.Start()
	for i := 0; i < testIterations; i++ {
		index := i % len(dataItems)
		item := dataItems[index]
		ads.GenerateProof(item.URL, item.Hash)
	}
	genDuration := timer.Stop()
	avgGenTime := genDuration / float64(testIterations)
	return avgGenTime
}

func testDynamicOperationsSimple(ads *auth.AuthenticatedDataStructure, dataItems []*types.DataItem, dataTypes []string, generator *dataset.DatasetGenerator) (float64, float64, float64) {
	addTime := testAddOperationSimple(ads, generator, dataTypes)

	updateTime := testUpdateOperationSimple(ads, dataItems)

	deleteTime := testDeleteOperationSimple(ads, dataItems)

	return addTime, updateTime, deleteTime
}

func testAddOperationSimple(ads *auth.AuthenticatedDataStructure, generator *dataset.DatasetGenerator, dataTypes []string) float64 {
	items := make([]*types.DataItem, testIterations)
	for i := 0; i < testIterations; i++ {
		newItem := generator.GenerateRandomDataItem(dataTypes)
		newItem.URL = fmt.Sprintf("%s_test_add_%d_%d", newItem.URL, time.Now().UnixNano(), i)
		items[i] = newItem
	}

	timer := utils.NewTimer("Add operation")
	var totalDuration float64 = 0
	for i := 0; i < testIterations; i++ {
		timer.Start()
		ads.InsertDataItem(items[i])
		opDuration := timer.Stop()
		totalDuration += opDuration
	}
	avgDuration := totalDuration / float64(testIterations)
	return avgDuration
}

func testUpdateOperationSimple(ads *auth.AuthenticatedDataStructure, dataItems []*types.DataItem) float64 {
	startIndex := len(dataItems) / 2

	timer := utils.NewTimer("Update operation")
	var totalDuration float64 = 0
	for i := 0; i < testIterations; i++ {
		idx := startIndex + (i % (len(dataItems) - startIndex))
		item := dataItems[idx]

		newContent := make([]byte, dataItemSize)
		rand.Read(newContent)
		newHash := utils.Keccak256(newContent)

		timer.Start()
		ads.UpdateDataItem(item.URL, newHash)
		opDuration := timer.Stop()
		totalDuration += opDuration
	}
	avgDuration := totalDuration / float64(testIterations)
	return avgDuration
}

func testDeleteOperationSimple(ads *auth.AuthenticatedDataStructure, dataItems []*types.DataItem) float64 {
	maxIndex := len(dataItems) / 2

	timer := utils.NewTimer("Delete operation")
	var totalDuration float64 = 0
	actualOperations := 0
	for i := 0; i < testIterations && i < maxIndex; i++ {
		item := dataItems[i]

		timer.Start()
		ads.DeleteDataItem(item.URL)
		opDuration := timer.Stop()
		totalDuration += opDuration
		actualOperations++
	}
	avgDuration := totalDuration / float64(actualOperations)
	return avgDuration
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64, avg float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		diff := v - avg
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(len(values)))
}

func emitOneProofJSON(ads *auth.AuthenticatedDataStructure, dataItems []*types.DataItem, index int) error {
	if len(dataItems) == 0 {
		return fmt.Errorf("no data item available for proof generation")
	}
	if index < 0 || index >= len(dataItems) {
		index = 0
	}

	item := dataItems[index]
	proof, err := ads.GenerateProof(item.URL, item.Hash)
	if err != nil {
		return fmt.Errorf("failed to generate proof: %v", err)
	}

	subtreeProofPath := make([]string, len(proof.MerkleProof.Siblings))
	for i, sib := range proof.MerkleProof.Siblings {
		subtreeProofPath[i] = fmt.Sprintf("0x%x", sib)
	}

	otherSubtreeRoots := []string{}
	dataTypeIndex := 0
	for i, dt := range ads.DataTypes {
		if dt == proof.DataType {
			dataTypeIndex = i
		}
	}
	for i, dt := range ads.DataTypes {
		if i == dataTypeIndex {
			continue
		}
		if root, ok := proof.OtherRoots[dt]; ok {
			otherSubtreeRoots = append(otherSubtreeRoots, fmt.Sprintf("0x%x", root))
		}
	}

	out := map[string]interface{}{
		"topRoot":           fmt.Sprintf("0x%x", proof.TopRoot),
		"dataHash":          fmt.Sprintf("0x%x", item.Hash),
		"url":               item.URL,
		"subtreeProofPath":  subtreeProofPath,
		"otherSubtreeRoots": otherSubtreeRoots,
		"dataTypeIndex":     dataTypeIndex,
		"leafIndex":         proof.MerkleProof.Index,
		"subtreeHeight":     len(proof.MerkleProof.Siblings),
		"typeCount":         len(ads.DataTypes),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
