package tool

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
	// 命令行参数定义
	dataPath   = flag.String("datapath", filepath.Join("./", "data"), "Path to your custom 'data' directory")
	datName    = flag.String("datname", "", "Name of the generated dat file")
	outputPath = flag.String("outputpath", "./publish", "Output path to the generated files")
	makeMode   = flag.String("mode", "", "Make geoip or geosite")
	directPath = flag.String("direct", "./domain_data/cn", "Path to the CN domain list file")
	proxyPath  = flag.String("proxy", "./domain_data/gfw", "Path to the GFW domain list file")
	customPath = flag.String("custom", "./custom.toml", "Path to the custom configuration file")
)

// Config 结构体用于解析 custom.toml 文件中的自定义规则。
type Config struct {
	Direct struct { // 直连列表 (Direct)
		Add    []string // 要添加的规则
		Remove []string // 要移除的规则
	}
	Proxy struct { // 代理列表 (Proxy)
		Add    []string
		Remove []string
	}
}

// read 读取指定路径文件的所有非空行，并返回一个字符串切片。
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
		if line != "" { // 仅处理非空行
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// RemoveData 从 targetFile 中移除 removeData 列表中的所有行。
func RemoveData(targetFile string, removeData []string) {
	// 读取源文件内容
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
			fmt.Printf("%s removed from %s\n", line, targetFile) // 打印被移除的行
		}
	}

	// 写入输出文件（覆盖源文件）
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

// AddData 将 addData 列表中的所有行追加到 targetFile 的末尾。
func AddData(targetFile string, addData []string) {
	// 读取源文件内容
	sourceData, err := read(targetFile)
	if err != nil {
		fmt.Printf("Error reading source file: %v\n", err)
		os.Exit(1)
	}

	// 追加新数据
	sourceData = append(sourceData, addData...)

	// 写入输出文件（覆盖源文件）
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

// RunTool 是程序的入口函数，根据 -mode 参数执行 geoip 或 geosite 的生成。
func RunTool() {
	flag.Parse()
	if len(*makeMode) == 0 {
		fmt.Println("-mode [geoip|geosite]")
		os.Exit(0)
	}

	if *makeMode == "geoip" {
		if *datName == "" {
			*datName = "geoip.dat"
		}
		geoip()      // 生成 geoip.dat
		gen_sha256() // 生成 SHA256 校验和文件
		os.Exit(0)
	}

	if *makeMode == "geosite" {
		if *datName == "" {
			*datName = "geosite.dat"
		}

		var config Config
		// 读取并解析 custom.toml 配置文件
		toml.DecodeFile(*customPath, &config)

		// 处理 Direct 列表的自定义规则（添加和移除）
		AddData(*directPath, config.Direct.Add)
		RemoveData(*directPath, config.Direct.Remove)

		// 处理 Proxy 列表的自定义规则（添加和移除）
		AddData(*proxyPath, config.Proxy.Add)
		RemoveData(*proxyPath, config.Proxy.Remove)

		geositeEntry() // 生成 geosite.dat
		gen_sha256()   // 生成 SHA256 校验和文件

		os.Exit(0)
	}

	fmt.Println("-mode geoip or -mode geosite")
}

// gen_sha256 为生成的 dat 文件计算 SHA256 校验和并写入同名文件（后缀为 .sha256sum）。
func gen_sha256() {
	file_path := filepath.Join(*outputPath, *datName)
	file, _ := os.ReadFile(file_path)
	sum := sha256.Sum256(file)
	// 格式：<SHA256 校验和>  <文件名>
	str := hex.EncodeToString(sum[:]) + "  " + *datName
	os.WriteFile(file_path+".sha256sum", []byte(str), 0644)
}
