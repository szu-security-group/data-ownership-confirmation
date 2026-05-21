package dataset

import (
	"crypto/rand"
	"data-ownership-system/pkg/types"
	"encoding/hex"
	"encoding/json"
	"fmt"
	mathrand "math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DatasetConfig struct {
	NumItems    int
	NumTypes    int
	ItemSize    int
	CompanyName string
	Protocol    string
}

type DatasetGenerator struct {
	config         *DatasetConfig
	dataDir        string
	rng            *mathrand.Rand
	lastTypeCounts map[string]int
}

func NewDatasetGenerator(config *DatasetConfig, dataDir string) *DatasetGenerator {
	source := mathrand.NewSource(time.Now().UnixNano())
	rng := mathrand.New(source)

	return &DatasetGenerator{
		config:  config,
		dataDir: dataDir,
		rng:     rng,
	}
}

func (dg *DatasetGenerator) GenerateDataset() ([]*types.DataItem, []string, error) {
	filename := dg.getDatasetFilename()
	filePath := filepath.Join(dg.dataDir, filename)

	if dg.datasetExists(filePath) {
		fmt.Printf("Dataset exists; loading from file: %s\n", filename)
		dataItems, dataTypes, err := dg.loadDatasetFromFile(filePath)
		if err != nil {
			return nil, nil, err
		}

		dg.calculateTypeCountsFromData(dataItems, dataTypes)

		return dataItems, dataTypes, nil
	}

	fmt.Printf("Generating new dataset: n=%d, m=%d, size=%d bytes\n",
		dg.config.NumItems, dg.config.NumTypes, dg.config.ItemSize)

	dataTypes := dg.generateDataTypes()

	dataItems := dg.generateDataItems(dataTypes)

	err := dg.saveDatasetToFile(filePath, dataItems, dataTypes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to save dataset: %v", err)
	}

	fmt.Printf("Dataset generated and saved to: %s\n", filename)

	return dataItems, dataTypes, nil
}

func (dg *DatasetGenerator) generateDataTypes() []string {
	dataTypes := make([]string, dg.config.NumTypes)

	typeNames := []string{
		"images", "videos", "documents", "audio", "databases",
		"logs", "configs", "reports", "analytics", "social",
		"financial", "medical", "educational", "research", "marketing",
		"sales", "inventory", "customer", "product", "sensor",
	}

	for i := 0; i < dg.config.NumTypes; i++ {
		if i < len(typeNames) {
			dataTypes[i] = typeNames[i]
		} else {
			dataTypes[i] = fmt.Sprintf("type_%d", i)
		}
	}

	return dataTypes
}

func (dg *DatasetGenerator) generateDataItems(dataTypes []string) []*types.DataItem {
	dataItems := make([]*types.DataItem, dg.config.NumItems)

	typeCounts := make(map[string]int)
	for _, dataType := range dataTypes {
		typeCounts[dataType] = 0
	}

	contentBuffer := make([]byte, dg.config.ItemSize)

	for i := 0; i < dg.config.NumItems; i++ {
		typeIndex := dg.rng.Intn(len(dataTypes))
		dataType := dataTypes[typeIndex]

		typeCounts[dataType]++

		url := dg.generateURL(dataType, i)

		_, err := rand.Read(contentBuffer)
		if err != nil {
			for j := range contentBuffer {
				contentBuffer[j] = byte(dg.rng.Intn(256))
			}
		}

		content := make([]byte, len(contentBuffer))
		copy(content, contentBuffer)
		item := types.NewDataItem(url, content)
		item.Content = nil
		dataItems[i] = item
	}

	dg.lastTypeCounts = typeCounts

	return dataItems
}

func (dg *DatasetGenerator) GetLastTypeCounts() map[string]int {
	return dg.lastTypeCounts
}

func (dg *DatasetGenerator) calculateTypeCountsFromData(dataItems []*types.DataItem, dataTypes []string) {
	typeCounts := make(map[string]int)
	for _, dataType := range dataTypes {
		typeCounts[dataType] = 0
	}

	for _, item := range dataItems {
		parts := strings.Split(item.URL, "/")
		if len(parts) >= 4 {
			dataType := parts[3]
			if _, exists := typeCounts[dataType]; exists {
				typeCounts[dataType]++
			}
		}
	}

	dg.lastTypeCounts = typeCounts
}

func (dg *DatasetGenerator) generateURL(dataType string, index int) string {
	filename := fmt.Sprintf("file_%d_%d", index, dg.rng.Intn(10000))
	return fmt.Sprintf("%s://%s/%s/%s",
		dg.config.Protocol, dg.config.CompanyName, dataType, filename)
}

func (dg *DatasetGenerator) generateRandomContent() []byte {
	content := make([]byte, dg.config.ItemSize)

	_, err := rand.Read(content)
	if err != nil {
		fmt.Printf("warning: crypto/rand failed; falling back to math/rand: %v\n", err)
		for i := range content {
			content[i] = byte(dg.rng.Intn(256))
		}
	}

	return content
}

func (dg *DatasetGenerator) getDatasetFilename() string {
	return fmt.Sprintf("dataset_n%d_m%d_s%d.json",
		dg.config.NumItems, dg.config.NumTypes, dg.config.ItemSize)
}

func (dg *DatasetGenerator) datasetExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

type DatasetFile struct {
	Config    *DatasetConfig `json:"config"`
	DataTypes []string       `json:"data_types"`
	Items     []DataItemFile `json:"items"`
}

type DataItemFile struct {
	URL     string `json:"url"`
	Content string `json:"content"`
	Hash    string `json:"hash"`
}

func (dg *DatasetGenerator) saveDatasetToFile(filePath string, dataItems []*types.DataItem, dataTypes []string) error {
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	fileData := &DatasetFile{
		Config:    dg.config,
		DataTypes: dataTypes,
		Items:     make([]DataItemFile, len(dataItems)),
	}

	for i, item := range dataItems {
		fileData.Items[i] = DataItemFile{
			URL:     item.URL,
			Content: "",
			Hash:    hex.EncodeToString(item.Hash),
		}
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(fileData)
	if err != nil {
		return fmt.Errorf("JSON serialization failed: %v", err)
	}

	return nil
}

func (dg *DatasetGenerator) loadDatasetFromFile(filePath string) ([]*types.DataItem, []string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var fileData DatasetFile
	err = decoder.Decode(&fileData)
	if err != nil {
		return nil, nil, fmt.Errorf("JSON deserialization failed: %v", err)
	}

	if !dg.configMatches(fileData.Config) {
		return nil, nil, fmt.Errorf("dataset configuration mismatch")
	}

	dataItems := make([]*types.DataItem, len(fileData.Items))
	for i, itemFile := range fileData.Items {
		var hashBytes []byte
		if len(itemFile.Hash) > 0 {
			var err error
			hashBytes, err = hex.DecodeString(itemFile.Hash)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse hash: %v", err)
			}
		}
		dataItems[i] = &types.DataItem{URL: itemFile.URL, Content: nil, Hash: hashBytes}
	}

	return dataItems, fileData.DataTypes, nil
}

func (dg *DatasetGenerator) configMatches(other *DatasetConfig) bool {
	return dg.config.NumItems == other.NumItems &&
		dg.config.NumTypes == other.NumTypes &&
		dg.config.ItemSize == other.ItemSize
}

func (dg *DatasetGenerator) generateRandomContentForIndex(index int) []byte {
	localRng := mathrand.New(mathrand.NewSource(int64(index)))

	content := make([]byte, dg.config.ItemSize)
	for i := range content {
		content[i] = byte(localRng.Intn(256))
	}

	return content
}

func (dg *DatasetGenerator) GenerateRandomDataItem(dataTypes []string) *types.DataItem {
	typeIndex := dg.rng.Intn(len(dataTypes))
	dataType := dataTypes[typeIndex]

	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("dynamic_%d_%d", timestamp, dg.rng.Intn(10000))
	url := fmt.Sprintf("%s://%s/%s/%s",
		dg.config.Protocol, dg.config.CompanyName, dataType, filename)

	content := dg.generateRandomContent()

	return types.NewDataItem(url, content)
}

func (dg *DatasetGenerator) SelectRandomExistingItem(dataItems []*types.DataItem) *types.DataItem {
	if len(dataItems) == 0 {
		return nil
	}

	index := dg.rng.Intn(len(dataItems))
	return dataItems[index]
}

func (dg *DatasetGenerator) String() string {
	return fmt.Sprintf("DatasetGenerator{n: %d, m: %d, size: %d}",
		dg.config.NumItems, dg.config.NumTypes, dg.config.ItemSize)
}
