package tool

import (
	"fmt"

	"os"
	"path/filepath"

	"google.golang.org/protobuf/proto"
)

// geositeEntry 是生成 geosite.dat 文件的入口函数。
func geositeEntry() {
	dir := GetDataDir() // 获取数据目录路径
	listInfoMap := make(ListInfoMap)

	// 遍历数据目录下的所有文件
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil // 跳过目录
		}
		// 处理单个文件，生成 ListInfo 并存入 listInfoMap
		if err := listInfoMap.Marshal(path); err != nil {
			return err
		}
		return nil
	}); err != nil {
		fmt.Println("Failed:", err)
		os.Exit(1)
	}

	// 展平包含（include）的列表，并为 Domain 类型的规则生成唯一列表（去重）
	if err := listInfoMap.FlattenAndGenUniqueDomainList(); err != nil {
		fmt.Println("Failed:", err)
		os.Exit(1)
	}

	// 预留的排除属性映射，目前为空
	excludeAttrsInFile := make(map[fileName]map[attribute]bool)

	// 将 ListInfoMap 转换为 router.GeoSiteList 结构
	if geositeList := listInfoMap.ToProto(excludeAttrsInFile); geositeList != nil {
		// 序列化为 Protobuf 字节
		protoBytes, err := proto.Marshal(geositeList)
		if err != nil {
			fmt.Println("Failed:", err)
			os.Exit(1)
		}

		// 创建输出目录
		if err := os.MkdirAll(*outputPath, 0755); err != nil {
			fmt.Println("Failed:", err)
			os.Exit(1)
		}

		// 写入 geosite.dat 文件
		if err := os.WriteFile(filepath.Join(*outputPath, *datName), protoBytes, 0644); err != nil {
			fmt.Println("Failed:", err)
			os.Exit(1)
		} else {
			fmt.Printf("%s has been generated successfully in '%s'.\n", *datName, *outputPath)
		}
	}

}
