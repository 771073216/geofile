package tool

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	router "github.com/xtls/xray-core/app/router"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/net"
)

// fileName 用于表示数据文件（如 CN, GFW 等）的名称类型
type fileName string

// attribute 用于表示规则属性（如 @cn, @ads 等）的类型
type attribute string

// GetDataDir 返回用于生成列表的 "data" 目录的路径。
// 查找顺序：
// 1. 用户在运行程序时设置的 `datapath` 选项。
// 2. 如果存在，使用默认路径 "./data"（当前工作目录下的 data 目录）。
// 3. 如果以上都不存在，使用 GOPATH 模式下 `v2fly/domain-list-community` 项目的 data 目录。
func GetDataDir() string {
	if *dataPath != "" { // 如果用户设置了 dataPath 选项，则使用它
		fmt.Printf("Use domain list files in '%s' directory.\n", *dataPath)
		return *dataPath
	}

	defaultDataDir := filepath.Join("./", "data")
	if _, err := os.Stat(defaultDataDir); !os.IsNotExist(err) { // 如果默认的 "./data" 目录存在，则使用它
		fmt.Printf("Use domain list files in '%s' directory.\n", defaultDataDir)
		return defaultDataDir
	}

	// 否则，使用 GOPATH 下的 v2fly/domain-list-community 项目的 data 目录
	return filepath.Join(GetGOPATH(), "src", "github.com", "v2fly", "domain-list-community", "data")
}

// envFile 返回 Go 环境配置文件的名称。
// 复制自 https://github.com/golang/go/.../cfg/cfg.go#L150-L166
func envFile() (string, error) {
	if file := os.Getenv("GOENV"); file != "" {
		if file == "off" {
			return "", fmt.Errorf("GOENV=off")
		}
		return file, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", fmt.Errorf("missing user-config dir")
	}
	return filepath.Join(dir, "go", "env"), nil
}

// GetRuntimeEnv 返回通过 `go env -w key=value` 设置的运行时环境变量的值。
func GetRuntimeEnv(key string) (string, error) {
	file, err := envFile()
	if err != nil {
		return "", err
	}
	if file == "" {
		return "", fmt.Errorf("missing runtime env file")
	}
	var data []byte
	var runtimeEnv string
	data, readErr := os.ReadFile(file)
	if readErr != nil {
		return "", readErr
	}
	envStrings := strings.Split(string(data), "\n")
	for _, envItem := range envStrings {
		envItem = strings.TrimSuffix(envItem, "\r")
		envKeyValue := strings.Split(envItem, "=")
		if strings.EqualFold(strings.TrimSpace(envKeyValue[0]), key) {
			runtimeEnv = strings.TrimSpace(envKeyValue[1])
		}
	}
	return runtimeEnv, nil
}

// GetGOPATH 返回 GOPATH 环境变量的值（字符串类型）。它不会为空。
func GetGOPATH() string {
	// 1. 用户显式设置的 GOPATH（如 export GOPATH=/path）
	GOPATH := os.Getenv("GOPATH")
	if GOPATH == "" {
		var err error
		// 2. 用户通过 `go env -w GOPATH=/path` 设置的 GOPATH
		GOPATH, err = GetRuntimeEnv("GOPATH")
		if err != nil {
			// 3. Golang 使用的默认值
			return build.Default.GOPATH
		}
		if GOPATH == "" {
			return build.Default.GOPATH
		}
		return GOPATH
	}
	return GOPATH
}

// isEmpty 检查一个已经去除空格的规则行是否为空
func isEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

// removeComment 移除规则行中的注释部分（从 # 字符开始）
func removeComment(line string) string {
	idx := strings.Index(line, "#")
	if idx == -1 {
		return line
	}
	return strings.TrimSpace(line[:idx])
}

// ParseIP 解析一个 CIDR 字符串（如 "192.168.1.1/24" 或 "::1/128"）
// 并将其转换为 router.CIDR 结构体
func ParseIP(s string) (*router.CIDR, error) {
	var addr, mask string
	i := strings.Index(s, "/")
	if i < 0 { // 如果没有斜杠，则表示没有掩码，默认使用完整的掩码（IPv4: /32, IPv6: /128）
		addr = s
	} else {
		addr = s[:i]
		mask = s[i+1:]
	}
	ip := net.ParseAddress(addr)
	switch ip.Family() {
	case net.AddressFamilyIPv4:
		bits := uint32(32) // IPv4 默认掩码是 /32
		if len(mask) > 0 {
			bits64, err := strconv.ParseUint(mask, 10, 32)
			if err != nil {
				return nil, errors.New("invalid network mask for router: ", mask).Base(err)
			}
			bits = uint32(bits64)
		}
		if bits > 32 {
			return nil, errors.New("invalid network mask for router: ", bits)
		}
		return &router.CIDR{
			Ip:     []byte(ip.IP()),
			Prefix: bits,
		}, nil
	case net.AddressFamilyIPv6:
		bits := uint32(128) // IPv6 默认掩码是 /128
		if len(mask) > 0 {
			bits64, err := strconv.ParseUint(mask, 10, 32)
			if err != nil {
				return nil, errors.New("invalid network mask for router: ", mask).Base(err)
			}
			bits = uint32(bits64)
		}
		if bits > 128 {
			return nil, errors.New("invalid network mask for router: ", bits)
		}
		return &router.CIDR{
			Ip:     []byte(ip.IP()),
			Prefix: bits,
		}, nil
	default:
		return nil, errors.New("unsupported address for router: ", s)
	}
}
