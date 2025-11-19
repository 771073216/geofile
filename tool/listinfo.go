package tool

import (
	"bufio"
	"errors"

	"os"
	"sort"
	"strings"

	router "github.com/xtls/xray-core/app/router"
)

// ListInfo 是数据目录下单个文件的信息结构。
// 它包含文件中所有类型的规则，以及为了方便后续处理而存储的相同规则的多种结构。
type ListInfo struct {
	Name                    fileName                       // 列表文件名称 (e.g., CN)
	HasInclusion            bool                           // 标记文件是否包含 `include:` 规则
	InclusionAttributeMap   map[fileName][]attribute       // 包含的文件名及其需要的属性 (e.g., {"GOOGLE": ["@cn", "@gfw"]})
	FullTypeList            []*router.Domain               // full 类型的规则列表
	KeywordTypeList         []*router.Domain               // keyword (plain) 类型的规则列表
	RegexpTypeList          []*router.Domain               // regexp 类型的规则列表
	AttributeRuleUniqueList []*router.Domain               // 带有属性规则的列表 (未去重)
	DomainTypeList          []*router.Domain               // domain 类型的规则列表
	DomainTypeUniqueList    []*router.Domain               // domain 类型的规则去重后的列表
	AttributeRuleListMap    map[attribute][]*router.Domain // 按属性分组的规则列表 (e.g., {"@cn": [...], "@ads": [...]})
	GeoSite                 *router.GeoSite                // 最终生成的 GeoSite 结构
}

// NewListInfo 返回一个初始化的 ListInfo 结构体。
func NewListInfo() *ListInfo {
	return &ListInfo{
		// 初始化所有 map 和 slice
		InclusionAttributeMap:   make(map[fileName][]attribute),
		FullTypeList:            make([]*router.Domain, 0, 10),
		KeywordTypeList:         make([]*router.Domain, 0, 10),
		RegexpTypeList:          make([]*router.Domain, 0, 10),
		AttributeRuleUniqueList: make([]*router.Domain, 0, 10),
		DomainTypeList:          make([]*router.Domain, 0, 10),
		DomainTypeUniqueList:    make([]*router.Domain, 0, 10),
		AttributeRuleListMap:    make(map[attribute][]*router.Domain),
	}
}

// ProcessList 逐行处理数据目录中单个文件，并生成该文件的 ListInfo。
func (l *ListInfo) ProcessList(file *os.File) error {
	scanner := bufio.NewScanner(file)
	// 逐行解析文件以生成 ListInfo
	for scanner.Scan() {
		line := scanner.Text()

		// 移除空行和注释
		if isEmpty(line) {
			continue
		}
		line = removeComment(line)
		if isEmpty(line) {
			continue
		}

		// 解析单条规则
		parsedRule, err := l.parseRule(line)
		if err != nil {
			return err
		}
		if parsedRule == nil {
			continue // 可能是 include 规则
		}

		// 对解析后的规则进行分类和存储
		l.classifyRule(parsedRule)
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// parseRule 解析一行文本，将其转换为 router.Domain 规则结构。
func (l *ListInfo) parseRule(line string) (*router.Domain, error) {
	line = strings.TrimSpace(line)

	if line == "" {
		return nil, errors.New("empty line")
	}

	// 首先解析 `include` 规则，例如: `include:google`, `include:google @cn @gfw`
	if strings.HasPrefix(line, "include:") {
		l.parseInclusion(line)
		return nil, nil // include 规则不返回 router.Domain
	}

	// 解析非 include 规则
	parts := strings.Split(line, " ")
	ruleWithType := strings.TrimSpace(parts[0]) // 规则主体 (如 domain:google.com)
	if ruleWithType == "" {
		return nil, errors.New("empty rule")
	}

	var rule router.Domain
	// 解析规则类型和值
	if err := l.parseTypeRule(ruleWithType, &rule); err != nil {
		return nil, err
	}

	// 解析后续的属性 (attributes)
	for _, attrString := range parts[1:] {
		if attrString = strings.TrimSpace(attrString); attrString != "" {
			attr, err := l.parseAttribute(attrString)
			if err != nil {
				return nil, err
			}
			rule.Attribute = append(rule.Attribute, attr)
		}
	}

	return &rule, nil
}

// parseInclusion 解析 `include:` 规则，并将包含信息添加到 ListInfo 中。
// 例如: `include:google @cn @gfw`
func (l *ListInfo) parseInclusion(inclusion string) {
	inclusionVal := strings.TrimPrefix(strings.TrimSpace(inclusion), "include:")
	l.HasInclusion = true
	inclusionValSlice := strings.Split(inclusionVal, "@")
	// 文件名转换为大写
	filename := fileName(strings.ToUpper(strings.TrimSpace(inclusionValSlice[0])))

	switch len(inclusionValSlice) {
	case 1: // 没有属性的包含规则，例如: `include:google`
		// 使用 '@' 作为占位符属性，表示包含该文件中的所有规则（无论有无属性）
		l.InclusionAttributeMap[filename] = append(l.InclusionAttributeMap[filename], attribute("@"))
	default: // 带有属性的包含规则，例如: `include:google @cn @gfw`
		// 遍历所有属性
		for _, attr := range inclusionValSlice[1:] {
			attr = strings.ToLower(strings.TrimSpace(attr))
			if attr != "" {
				// 属性以 "@" 字符开头存储，例如: '@cn'
				l.InclusionAttributeMap[filename] = append(l.InclusionAttributeMap[filename], attribute("@"+attr))
			}
		}
	}
}

// parseTypeRule 解析规则类型和值，例如 "domain:google.com" 或 "google.com"
func (l *ListInfo) parseTypeRule(domain string, rule *router.Domain) error {
	kv := strings.Split(domain, ":")
	switch len(kv) {
	case 1: // 没有类型前缀的行，默认视为 domain 类型
		rule.Type = router.Domain_Domain
		rule.Value = strings.ToLower(strings.TrimSpace(kv[0]))
	case 2: // 带有类型前缀的行
		ruleType := strings.TrimSpace(kv[0])
		ruleVal := strings.TrimSpace(kv[1])
		rule.Value = strings.ToLower(ruleVal) // 规则值转小写（regexp 除外）
		switch strings.ToLower(ruleType) {
		case "full":
			rule.Type = router.Domain_Full
		case "domain":
			rule.Type = router.Domain_Domain
		case "keyword":
			rule.Type = router.Domain_Plain // Plain 对应 keyword
		case "regexp":
			rule.Type = router.Domain_Regex
			rule.Value = ruleVal // 正则表达式规则值保留原始大小写
		default:
			return errors.New("unknown domain type: " + ruleType)
		}
	}
	return nil
}

// parseAttribute 解析属性字符串（例如 "@cn"）并转换为 router.Domain_Attribute 结构。
func (l *ListInfo) parseAttribute(attr string) (*router.Domain_Attribute, error) {
	if attr[0] != '@' {
		return nil, errors.New("invalid attribute: " + attr)
	}
	attr = attr[1:] // 移除属性前缀 `@` 字符

	var attribute router.Domain_Attribute
	attribute.Key = strings.ToLower(attr) // 属性键转小写
	// 属性的值固定为布尔值 true
	attribute.TypedValue = &router.Domain_Attribute_BoolValue{BoolValue: true}
	return &attribute, nil
}

// classifyRule 对单个规则进行分类，并写入 ListInfo 结构。
func (l *ListInfo) classifyRule(rule *router.Domain) {
	// 规则分类逻辑：优先判断是否有属性
	if len(rule.Attribute) > 0 {
		// 带有属性的规则
		l.AttributeRuleUniqueList = append(l.AttributeRuleUniqueList, rule)
		var attrsString attribute
		// 构造一个包含所有属性的字符串作为 map 的键，例如 "@cn@ads"
		for _, attr := range rule.Attribute {
			attrsString += attribute("@" + attr.GetKey())
		}
		l.AttributeRuleListMap[attrsString] = append(l.AttributeRuleListMap[attrsString], rule)
	} else {
		// 不带属性的规则，按类型分类
		switch rule.Type {
		case router.Domain_Full:
			l.FullTypeList = append(l.FullTypeList, rule)
		case router.Domain_Domain:
			l.DomainTypeList = append(l.DomainTypeList, rule)
		case router.Domain_Plain: // keyword
			l.KeywordTypeList = append(l.KeywordTypeList, rule)
		case router.Domain_Regex:
			l.RegexpTypeList = append(l.RegexpTypeList, rule)
		}
	}
}

// Flatten 展平文件中的 `include` 规则，将所需规则添加到当前 ListInfo 中。
// 它还为 domain 类型的规则生成一个域名前缀树（trie）以进行去重。
func (l *ListInfo) Flatten(lm *ListInfoMap) error {
	if l.HasInclusion {
		// 遍历所有包含的列表文件及其属性
		for filename, attrs := range l.InclusionAttributeMap {
			for _, attrWanted := range attrs {
				includedList := (*lm)[filename] // 被包含的 ListInfo

				switch string(attrWanted) {
				case "@": // 包含所有规则（包括无属性和有属性的）
					// 直接追加无属性的规则列表
					l.FullTypeList = append(l.FullTypeList, includedList.FullTypeList...)
					l.DomainTypeList = append(l.DomainTypeList, includedList.DomainTypeList...)
					l.KeywordTypeList = append(l.KeywordTypeList, includedList.KeywordTypeList...)
					l.RegexpTypeList = append(l.RegexpTypeList, includedList.RegexpTypeList...)

					// 直接追加所有带属性的规则
					l.AttributeRuleUniqueList = append(l.AttributeRuleUniqueList, includedList.AttributeRuleUniqueList...)
					for attr, domainList := range includedList.AttributeRuleListMap {
						l.AttributeRuleListMap[attr] = append(l.AttributeRuleListMap[attr], domainList...)
					}

				default: // 包含带有特定属性的规则，例如 "@cn"
					for attr, domainList := range includedList.AttributeRuleListMap {
						// 检查被包含规则的属性是否包含 attrWanted。
						// 为了处理多属性规则（如 "@cn@ads"），使用 "@" 作为分隔符进行判断。
						if strings.Contains(string(attr)+"@", string(attrWanted)+"@") {
							// 追加到当前列表的 AttributeRuleListMap 和 AttributeRuleUniqueList 中
							l.AttributeRuleListMap[attr] = append(l.AttributeRuleListMap[attr], domainList...)
							l.AttributeRuleUniqueList = append(l.AttributeRuleUniqueList, domainList...)
						}
					}
				}
			}
		}
	}

	// 对 domain 类型的规则进行排序，使得子域（点号更多）排在前面
	sort.Slice(l.DomainTypeList, func(i, j int) bool {
		return len(strings.Split(l.DomainTypeList[i].GetValue(), ".")) < len(strings.Split(l.DomainTypeList[j].GetValue(), "."))
	})

	// 使用 DomainTrie 排除 domain 类型的重复项（包括子域被更短的父域覆盖的情况）
	trie := NewDomainTrie()
	for _, domain := range l.DomainTypeList {
		// 插入 trie，如果成功（即不是重复或子域），则加入到 DomainTypeUniqueList
		success, err := trie.Insert(domain.GetValue())
		if err != nil {
			return err
		}
		if success {
			l.DomainTypeUniqueList = append(l.DomainTypeUniqueList, domain)
		}
	}

	return nil
}

// ToGeoSite 将 ListInfo 转换为 router.GeoSite 结构。
// 它还会根据用户指定的排除属性（excludeAttrs）排除特定规则。
func (l *ListInfo) ToGeoSite(excludeAttrs map[fileName]map[attribute]bool) {
	geosite := new(router.GeoSite)
	geosite.CountryCode = string(l.Name)

	// 拼接无属性规则（DomainTypeUniqueList 已去重）
	geosite.Domain = append(geosite.Domain, l.FullTypeList...)
	geosite.Domain = append(geosite.Domain, l.DomainTypeUniqueList...)
	geosite.Domain = append(geosite.Domain, l.RegexpTypeList...)

	// 添加 keyword 规则（过滤掉空值）
	for _, keywordRule := range l.KeywordTypeList {
		if len(strings.TrimSpace(keywordRule.GetValue())) > 0 {
			geosite.Domain = append(geosite.Domain, keywordRule)
		}
	}

	// 处理带有属性的规则
	if excludeAttrs != nil && excludeAttrs[l.Name] != nil {
		// 如果当前文件有排除属性设置
		excludeAttrsMap := excludeAttrs[l.Name]
		for _, domain := range l.AttributeRuleUniqueList {
			ifKeep := true
			// 检查规则的属性是否在排除列表中
			for _, attr := range domain.GetAttribute() {
				if excludeAttrsMap[attribute(attr.GetKey())] {
					ifKeep = false
					break
				}
			}
			if ifKeep {
				geosite.Domain = append(geosite.Domain, domain)
			}
		}
	} else {
		// 当前文件没有排除属性设置，直接添加所有带属性的规则
		geosite.Domain = append(geosite.Domain, l.AttributeRuleUniqueList...)
	}
	l.GeoSite = geosite
}
