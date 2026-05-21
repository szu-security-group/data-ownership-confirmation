package preprocessor

import (
	"data-ownership-system/utils"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type DatasetConfig struct {
	NumItems    int    `json:"NumItems"`
	NumTypes    int    `json:"NumTypes"`
	ItemSize    int    `json:"ItemSize"`
	CompanyName string `json:"CompanyName"`
	Protocol    string `json:"Protocol"`
}

type DataItem struct {
	URL     string `json:"url"`
	Content string `json:"content"`
	Hash    string `json:"hash"`
}

type Dataset struct {
	Config    DatasetConfig `json:"config"`
	DataTypes []string      `json:"data_types"`
	Items     []DataItem    `json:"items"`
}

type Statistics struct {
	TotalFiles   int
	TotalTypes   int
	TypeCounts   map[string]int
	AvgFileSize  float64
	TotalSize    int64
	LargestFile  string
	LargestSize  int64
	SmallestFile string
	SmallestSize int64
}

func PreprocessRealData(dataPath, protocol, companyName, outputFile string) error {
	if dataPath == "" {
		return fmt.Errorf("error: data path is required")
	}

	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		return fmt.Errorf("error: data path does not exist: %s", dataPath)
	}

	fmt.Println("=== Real dataset preprocessing started ===")
	fmt.Printf("Data path: %s\n", dataPath)
	fmt.Printf("Protocol: %s\n", protocol)
	fmt.Printf("Company: %s\n", companyName)
	fmt.Printf("Output file: %s\n", outputFile)
	fmt.Println()

	company := strings.TrimSuffix(companyName, "/")
	if company != companyName {
		fmt.Printf("Note: company name normalized to: %s\n", company)
		fmt.Println()
	}

	fmt.Println("Step 1: scan data directory...")
	dataTypes, filesByType, stats, err := scanDataDirectory(dataPath)
	if err != nil {
		return fmt.Errorf("failed to scan data directory: %v", err)
	}

	fmt.Println("Step 2: generate data items and compute hashes...")
	items, err := generateDataItems(filesByType, protocol, company, dataPath)
	if err != nil {
		return fmt.Errorf("failed to generate data items: %v", err)
	}

	dataset := Dataset{
		Config: DatasetConfig{
			NumItems:    len(items),
			NumTypes:    len(dataTypes),
			ItemSize:    int(stats.AvgFileSize),
			CompanyName: company,
			Protocol:    protocol,
		},
		DataTypes: dataTypes,
		Items:     items,
	}

	fmt.Println("Step 3: save JSON file...")
	outputPath := filepath.Join("data", outputFile)
	err = saveDataset(dataset, outputPath)
	if err != nil {
		return fmt.Errorf("failed to save dataset: %v", err)
	}

	printStatistics(stats, outputPath)

	fmt.Println("=== Real dataset preprocessing completed ===")

	return nil
}

func scanDataDirectory(rootPath string) ([]string, map[string][]string, *Statistics, error) {
	dataTypes := make(map[string]bool)
	filesByType := make(map[string][]string)
	stats := &Statistics{
		TypeCounts:   make(map[string]int),
		SmallestSize: int64(^uint64(0) >> 1),
	}

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == rootPath {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}

		pathParts := strings.Split(filepath.ToSlash(relPath), "/")
		if len(pathParts) < 2 {
			return fmt.Errorf("invalid file path format: %s", relPath)
		}

		dataType := pathParts[0]

		dataTypes[dataType] = true
		filesByType[dataType] = append(filesByType[dataType], path)

		fileInfo, err := d.Info()
		if err != nil {
			return err
		}

		fileSize := fileInfo.Size()
		stats.TotalFiles++
		stats.TotalSize += fileSize
		stats.TypeCounts[dataType]++

		if fileSize > stats.LargestSize {
			stats.LargestSize = fileSize
			stats.LargestFile = path
		}
		if fileSize < stats.SmallestSize {
			stats.SmallestSize = fileSize
			stats.SmallestFile = path
		}

		return nil
	})

	if err != nil {
		return nil, nil, nil, err
	}

	sortedTypes := make([]string, 0, len(dataTypes))
	for dataType := range dataTypes {
		sortedTypes = append(sortedTypes, dataType)
	}
	sort.Strings(sortedTypes)

	stats.TotalTypes = len(sortedTypes)
	if stats.TotalFiles > 0 {
		stats.AvgFileSize = float64(stats.TotalSize) / float64(stats.TotalFiles)
	}

	return sortedTypes, filesByType, stats, nil
}

func generateDataItems(filesByType map[string][]string, protocol, company, rootPath string) ([]DataItem, error) {
	var items []DataItem

	totalFiles := 0
	for _, paths := range filesByType {
		totalFiles += len(paths)
	}
	processedFiles := 0
	startTime := time.Now()

	fmt.Println("Start processing files and computing hashes...")
	fmt.Printf("Total files: %d\n", totalFiles)

	for dataType, paths := range filesByType {
		fmt.Printf("\nProcessing data type: %s (%dfiles)\n", dataType, len(paths))

		typeProcessed := 0

		for _, filePath := range paths {
			relPath, err := filepath.Rel(rootPath, filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to get relative path %s: %v", filePath, err)
			}

			urlPath := filepath.ToSlash(relPath)

			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %v", filePath, err)
			}

			hash := utils.Keccak256(content)

			item := DataItem{
				URL:     fmt.Sprintf("%s://%s/%s", protocol, company, urlPath),
				Content: "",
				Hash:    hex.EncodeToString(hash),
			}
			items = append(items, item)

			processedFiles++
			typeProcessed++

			if typeProcessed%10 == 0 || typeProcessed == len(paths) {
				elapsed := time.Since(startTime)
				filesPerSecond := float64(processedFiles) / elapsed.Seconds()
				remainingFiles := totalFiles - processedFiles
				remainingTime := time.Duration(float64(remainingFiles)/filesPerSecond) * time.Second

				fmt.Printf("\r%s: %d/%d (%.1f%%) | overall: %d/%d (%.1f%%) | %.1ffiles/s | ETA: %v",
					dataType, typeProcessed, len(paths), float64(typeProcessed)/float64(len(paths))*100,
					processedFiles, totalFiles, float64(processedFiles)/float64(totalFiles)*100,
					filesPerSecond, remainingTime.Round(time.Second))
			}
		}
		fmt.Println()
	}

	totalTime := time.Since(startTime)
	avgSpeed := float64(totalFiles) / totalTime.Seconds()
	fmt.Printf("\nProcessing completed; processed %d files in %v (average %.1f files/s)\n",
		totalFiles, totalTime.Round(time.Second), avgSpeed)

	return items, nil
}

func saveDataset(dataset Dataset, outputPath string) error {
	err := os.MkdirAll(filepath.Dir(outputPath), 0755)
	if err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(dataset)
	if err != nil {
		return err
	}

	return nil
}

func printStatistics(stats *Statistics, outputPath string) {
	fmt.Println("\n=== Dataset statistics ===")
	fmt.Printf("Total files: %d\n", stats.TotalFiles)
	fmt.Printf("Data types: %d\n", stats.TotalTypes)
	fmt.Printf("Total data size: %.2f MB\n", float64(stats.TotalSize)/(1024*1024))
	fmt.Printf("Average file size: %.2f KB\n", stats.AvgFileSize/1024)

	if stats.LargestFile != "" {
		fmt.Printf("Largest file: %s (%.2f KB)\n", stats.LargestFile, float64(stats.LargestSize)/1024)
	}
	if stats.SmallestFile != "" {
		fmt.Printf("Smallest file: %s (%.2f KB)\n", stats.SmallestFile, float64(stats.SmallestSize)/1024)
	}

	fmt.Println("\nFiles per type:")
	var types []string
	for dataType := range stats.TypeCounts {
		types = append(types, dataType)
	}
	sort.Strings(types)

	for _, dataType := range types {
		count := stats.TypeCounts[dataType]
		percentage := float64(count) / float64(stats.TotalFiles) * 100
		fmt.Printf("  %s: %d (%.1f%%)\n", dataType, count, percentage)
	}

	fmt.Printf("\nOutput file: %s\n", outputPath)
	if fileInfo, err := os.Stat(outputPath); err == nil {
		fmt.Printf("Output file size: %.2f KB\n", float64(fileInfo.Size())/1024)
	}
}
