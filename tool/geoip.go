package tool

import (
	"bufio"
	"fmt"
	"io"

	"os"
	"path/filepath"
	"strings"

	merge "github.com/EvilSuperstars/go-cidrman"
	router "github.com/xtls/xray-core/app/router"
	"google.golang.org/protobuf/proto"
)

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

func getCidrPerFile(dataDirMap map[string][]*router.CIDR) error {
	walkErr := filepath.Walk(*dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		filename := filepath.Base(path)
		fileExt := filepath.Ext(path)
		onlyFileName := strings.ToUpper(strings.TrimSuffix(filename, fileExt))
		cidrContainer := make([]*router.CIDR, 0)
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

func readFileLineByLine(path string, container *[]*router.CIDR) error {
	var str []string
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
		str = append(str, string(a))
	}
	merged_CIDR, _ := merge.MergeCIDRs(str)
	for _, cidr := range merged_CIDR {
		cidr1, err := ParseIP(cidr)
		if err != nil {
			fmt.Println(err)
		}
		*container = append(*container, cidr1)
	}

	return nil
}

func private(dataDirMap map[string][]*router.CIDR) {
	container1 := make([]*router.CIDR, 0)
	for _, cidr := range privateCIDRs {
		cidr1, err := ParseIP(cidr)
		if err != nil {
			fmt.Println(err)
		}
		container1 = append(container1, cidr1)
	}
	dataDirMap["PRIVATE"] = container1
}

func geoip() {
	cidrList := make(map[string][]*router.CIDR)
	private(cidrList)
	if err := getCidrPerFile(cidrList); err != nil {
		fmt.Println("Error looping data directory:", err)
		os.Exit(1)
	}
	geoIPList := new(router.GeoIPList)
	for cc, cidr := range cidrList {
		geoIPList.Entry = append(geoIPList.Entry, &router.GeoIP{
			CountryCode: cc,
			Cidr:        cidr,
		})
	}
	geoIPBytes, err := proto.Marshal(geoIPList)
	if err != nil {
		fmt.Println("Error marshalling geoip list:", err)
		os.Exit(1)
	}

	// Create output directory if not exist
	if _, err := os.Stat(*outputPath); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(*outputPath, 0755); mkErr != nil {
			fmt.Println("Failed: ", mkErr)
			os.Exit(1)
		}
	}

	if err := os.WriteFile(filepath.Join(*outputPath, *datName), geoIPBytes, 0644); err != nil {
		fmt.Println("Error writing geoip to file:", err)
		os.Exit(1)
	} else {
		fmt.Println(*datName, "has been generated successfully.")
	}
}
