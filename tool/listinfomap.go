package tool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	router "github.com/xtls/xray-core/app/router"
)

// ListInfoMap 是数据目录中文件及其对应 ListInfo 的映射。
type ListInfoMap map[fileName]*ListInfo

// Marshal 处理数据目录中的一个文件，为其生成并返回 ListInfo。
func (lm *ListInfoMap) Marshal(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	list := NewListInfo()
	// 文件名（去除路径，转大写）作为 ListInfo 的名称
	listName := fileName(strings.ToUpper(filepath.Base(path)))
	list.Name = listName

	// 处理文件内容，填充 ListInfo
	if err := list.ProcessList(file); err != nil {
		return err
	}

	(*lm)[listName] = list
	return nil
}

// FlattenAndGenUniqueDomainList 展平包含（include）的列表，并为每个文件的 domain 类型规则生成
// 唯一的（去重后的）域名列表。
func (lm *ListInfoMap) FlattenAndGenUniqueDomainList() error {
	// Inclusion Level 算法用于按依赖顺序（level）处理文件，确保被包含的列表先于包含它的列表被展平。
	inclusionLevel := make([]map[fileName]bool, 0, 20) // 存储每个级别的文件名
	okayList := make(map[fileName]bool)                // 存储已处理/可以处理的文件名
	inclusionLevelAllLength, loopTimes := 0, 0         // 已处理文件的总数和循环次数

	// 循环直到所有文件都被分级
	for inclusionLevelAllLength < len(*lm) {
		inclusionMap := make(map[fileName]bool)

		if loopTimes == 0 {
			// Level 1: 处理所有不包含任何其他文件的列表
			for _, listinfo := range *lm {
				if listinfo.HasInclusion {
					continue
				}
				inclusionMap[listinfo.Name] = true
			}
		} else {
			// Level 2+ : 处理所有依赖（包含）的文件都已在 okayList 中的列表
			for _, listinfo := range *lm {
				// 跳过没有包含规则或已经处理过的文件
				if !listinfo.HasInclusion || okayList[listinfo.Name] {
					continue
				}

				var passTimes int
				// 检查所有它包含的文件是否都已处理
				for filename := range listinfo.InclusionAttributeMap {
					if !okayList[filename] {
						break // 依赖的文件未处理，跳出
					}
					passTimes++
				}

				// 如果所有依赖的文件都已处理，则当前文件可以被处理
				if passTimes == len(listinfo.InclusionAttributeMap) {
					inclusionMap[listinfo.Name] = true
				}
			}
		}

		// 将当前级别的所有文件添加到 okayList
		for filename := range inclusionMap {
			okayList[filename] = true
		}

		inclusionLevel = append(inclusionLevel, inclusionMap)
		inclusionLevelAllLength += len(inclusionMap)
		loopTimes++
	}

	// 按照确定的依赖级别顺序进行展平（Flatten）操作
	for idx, inclusionMap := range inclusionLevel {
		fmt.Printf("Level %d:\n", idx+1)
		fmt.Println(inclusionMap)
		fmt.Println()

		for inclusionFilename := range inclusionMap {
			// 对当前级别的每个文件执行展平操作
			if err := (*lm)[inclusionFilename].Flatten(lm); err != nil {
				return err
			}
		}
	}

	return nil
}

// ToProto 为 ListInfoMap 中的每个文件生成一个 router.GeoSite 结构，
// 并返回一个包含所有 GeoSite 的 router.GeoSiteList 结构。
func (lm *ListInfoMap) ToProto(excludeAttrs map[fileName]map[attribute]bool) *router.GeoSiteList {
	protoList := new(router.GeoSiteList)
	for _, listinfo := range *lm {
		// 将 ListInfo 转换为 GeoSite 结构
		listinfo.ToGeoSite(excludeAttrs)
		// 将生成的 GeoSite 添加到 GeoSiteList 中
		protoList.Entry = append(protoList.Entry, listinfo.GeoSite)
	}
	return protoList
}
