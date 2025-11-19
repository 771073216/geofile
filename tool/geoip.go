package tool

import (
	"bufio"
	"fmt"
	"io"

	"os"
	"path/filepath"
	"strings"

	router "github.com/xtls/xray-core/app/router"
	"google.golang.org/protobuf/proto"
)

// privateCIDRs 是一个包含所有私有、保留或特殊用途 IP CIDR 范围的列表。
var privateCIDRs = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"255.255.255.255/32",
	"::/128",
	"::1/128",
	"fc00::/7",
	"ff00::/8",
	"fe80::/10",
}

// getCidrPerFile 遍历 data 目录下的文件，并为每个文件读取其包含的 CIDR 规则。
// 文件的基础名称（大写）作为键，存储在 dataDirMap 中。
func getCidrPerFile(dataDirMap map[string][]*router.CIDR) error {
	walkErr := filepath.Walk(*dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil // 跳过目录
		}

		filename := filepath.Base(path)
		fileExt := filepath.Ext(path)
		// 文件名（不含扩展名，转大写）作为国家/地区代码
		onlyFileName := strings.ToUpper(strings.TrimSuffix(filename, fileExt))
		cidrContainer := make([]*router.CIDR, 0)

		// 读取文件内容并解析为 CIDR 列表
		if err = readFileLineByLine(path, &cidrContainer); err != nil {
			return err
		}
		dataDirMap[onlyFileName] = cidrContainer
		return nil
	})

	if walkErr != nil {
		return walkErr
	}
	return nil
}

// readFileLineByLine 按行读取指定路径的文件，将每行解析为一个 router.CIDR 并追加到 container 中。
func readFileLineByLine(path string, container *[]*router.CIDR) error {
	var cidrStr []string
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	li := bufio.NewReader(file)
	for {
		a, _, c := li.ReadLine()
		if c == io.EOF {
			break
		}
		// 收集所有非 EOF 行的字符串
		cidrStr = append(cidrStr, string(a))
	}

	// 遍历收集到的字符串，解析为 router.CIDR 结构
	for _, cidr := range cidrStr {
		cidr1, err := ParseIP(cidr)
		if err != nil {
			fmt.Println(err) // 打印错误，但不终止流程
		}
		*container = append(*container, cidr1)
	}

	return nil
}

// private 将所有私有 CIDR 范围添加到 dataDirMap 中，使用 "PRIVATE" 作为键。
func private(dataDirMap map[string][]*router.CIDR) {
	container1 := make([]*router.CIDR, 0)
	for _, cidr := range privateCIDRs {
		cidr1, err := ParseIP(cidr)
		if err != nil {
			fmt.Println(err) // 打印错误，但不终止流程
		}
		container1 = append(container1, cidr1)
	}
	dataDirMap["PRIVATE"] = container1
}

// geoip 是生成 geoip.dat 文件的核心函数。
func geoip() {
	cidrList := make(map[string][]*router.CIDR)
	private(cidrList) // 首先添加私有 IP 范围

	// 读取 data 目录下的文件，收集所有 CIDR 规则
	if err := getCidrPerFile(cidrList); err != nil {
		fmt.Println("Error looping data directory:", err)
		os.Exit(1)
	}

	geoIPList := new(router.GeoIPList)
	// 将 map 中的数据转换为 router.GeoIPList 结构
	for cc, cidr := range cidrList {
		geoIPList.Entry = append(geoIPList.Entry, &router.GeoIP{
			CountryCode: cc, // 使用文件名（大写）作为国家/地区代码
			Cidr:        cidr,
		})
	}

	// 序列化为 Protobuf 字节
	geoIPBytes, err := proto.Marshal(geoIPList)
	if err != nil {
		fmt.Println("Error marshalling geoip list:", err)
		os.Exit(1)
	}

	// 检查并创建输出目录
	if _, err := os.Stat(*outputPath); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(*outputPath, 0755); mkErr != nil {
			fmt.Println("Failed: ", mkErr)
			os.Exit(1)
		}
	}

	// 写入 geoip.dat 文件
	if err := os.WriteFile(filepath.Join(*outputPath, *datName), geoIPBytes, 0644); err != nil {
		fmt.Println("Error writing geoip to file:", err)
		os.Exit(1)
	} else {
		fmt.Println(*datName, "has been generated successfully.")
	}
}
