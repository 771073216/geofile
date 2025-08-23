package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"strings"

	"os"
	"path/filepath"

	toml "github.com/BurntSushi/toml"
)

var (
	dataPath   = flag.String("datapath", filepath.Join("./", "data"), "Path to your custom 'data' directory")
	datName    = flag.String("datname", "", "Name of the generated dat file")
	outputPath = flag.String("outputpath", "./publish", "Output path to the generated files")
	makeMode   = flag.String("mode", "", "Make geoip or geosite")
	directPath = flag.String("direct", "./domain_data/cn", "Output path to the generated files")
	proxyPath  = flag.String("proxy", "./domain_data/gfw", "Output path to the generated files")
	customPath = flag.String("custom", "./custom.toml", "Output path to the generated files")
)

type Config struct {
	Direct struct {
		Add    []string
		Remove []string
	}
	Proxy struct {
		Add    []string
		Remove []string
	}
}

func read(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func removeData(targetFile string, removeData []string) {
	// 读取源文件
	sourceData, err := read(targetFile)
	if err != nil {
		fmt.Printf("Error reading source file: %v\n", err)
		os.Exit(1)
	}

	// 创建映射以便快速查找要移除的行
	removeMap := make(map[string]bool)
	for _, line := range removeData {
		removeMap[line] = true
	}

	// 过滤源数据
	var result []string
	for _, line := range sourceData {
		if !removeMap[line] {
			result = append(result, line)
		} else {
			fmt.Printf("%s not in %s\n", line, targetFile)
		}
	}

	// 写入输出文件
	outputFileHandle, err := os.Create(targetFile)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outputFileHandle.Close()

	writer := bufio.NewWriter(outputFileHandle)
	for _, line := range result {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			fmt.Printf("Error writing to output file: %v\n", err)
			os.Exit(1)
		}
	}
	writer.Flush()

	fmt.Printf("Successfully processed. Output written to %s\n", targetFile)
}

func addData(targetFile string, addData []string) {
	// 读取源文件
	sourceData, err := read(targetFile)
	if err != nil {
		fmt.Printf("Error reading source file: %v\n", err)
		os.Exit(1)
	}

	sourceData = append(sourceData, addData...)

	// 写入输出文件
	outputFileHandle, err := os.Create(targetFile)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outputFileHandle.Close()

	writer := bufio.NewWriter(outputFileHandle)
	for _, line := range sourceData {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			fmt.Printf("Error writing to output file: %v\n", err)
			os.Exit(1)
		}
	}
	writer.Flush()

	fmt.Printf("Successfully processed. Output written to %s\n", targetFile)
}

func main() {
	flag.Parse()
	if len(*makeMode) == 0 {
		fmt.Println("-mode [geoip|geosite]")
		os.Exit(0)
	}
	if *makeMode == "geoip" {
		if *datName == "" {
			*datName = "geoip.dat"
		}
		geoip()
		gen_sha256()
		os.Exit(0)
	}
	if *makeMode == "geosite" {
		if *datName == "" {
			*datName = "geosite.dat"
		}
		var config Config
		toml.DecodeFile(*customPath, &config)
		addData(*directPath, config.Direct.Add)
		removeData(*directPath, config.Direct.Remove)

		addData(*proxyPath, config.Proxy.Add)
		removeData(*proxyPath, config.Proxy.Remove)

		geositeEntry()
		gen_sha256()

		os.Exit(0)
	}
	fmt.Println("-mode geoip or -mode geosite")
}

func gen_sha256() {
	file_path := filepath.Join(*outputPath, *datName)
	file, _ := os.ReadFile(file_path)
	sum := sha256.Sum256(file)
	str := hex.EncodeToString(sum[:]) + "  " + *datName
	os.WriteFile(file_path+".sha256sum", []byte(str), 0644)
}
