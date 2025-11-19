package tool

import (
	"errors"
	"strings"
)

// node 表示 DomainTrie 中的一个节点。
type node struct {
	leaf     bool             // 标记此节点是否是一个完整的域名或域名的组件
	children map[string]*node // 存储子节点，键是域名的一部分（如 "com", "google"）
}

// newNode 创建并返回一个新的节点。
func newNode() *node {
	return &node{
		leaf:     false,
		children: make(map[string]*node),
	}
}

// getChild 获取指定字符串对应的子节点。
func (n *node) getChild(s string) *node {
	return n.children[s]
}

// hasChild 检查是否存在指定字符串对应的子节点。
func (n *node) hasChild(s string) bool {
	return n.getChild(s) != nil
}

// addChild 添加一个子节点。
func (n *node) addChild(s string, child *node) {
	n.children[s] = child
}

// isLeaf 检查当前节点是否被标记为叶子节点（即是否代表一个完整的域名规则）。
func (n *node) isLeaf() bool {
	return n.leaf
}

// DomainTrie 是用于 domain 类型规则的域名前缀树。
// 用于在包含（include）操作后对规则进行去重，并遵循最长匹配原则（父域优先）。
type DomainTrie struct {
	root *node
}

// NewDomainTrie 创建并返回一个新的域名前缀树。
func NewDomainTrie() *DomainTrie {
	return &DomainTrie{
		root: newNode(),
	}
}

// Insert 将一个域名规则字符串插入到域名前缀树中。
// 域名是按从右到左（从 TLD 到子域）的顺序插入的。
// 例如 "www.google.com" 的插入顺序是 "com" -> "google" -> "www"。
//
// 成功插入返回 true；如果域名为空或它已被其父域/自身覆盖，则返回 false。
func (t *DomainTrie) Insert(domain string) (bool, error) {
	if domain == "" {
		return false, errors.New("empty domain")
	}
	// 将域名按 "." 分割
	parts := strings.Split(domain, ".")

	node := t.root
	// 从后向前遍历域名组件 (e.g., com, google, www)
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]

		// 检查当前节点是否已被标记为一个完整的规则（叶子节点）。
		// 如果是，则当前域名是另一个已存在域名的子域，无需插入，返回 false。
		// 例如：已插入 "google.com"（node.leaf=true），现在插入 "www.google.com"，
		// 当遍历到 "google" 时，会发现 "google" 对应的节点是叶子节点，因此 "www.google.com" 不被插入。
		if node.isLeaf() {
			return false, nil
		}

		// 如果没有对应的子节点，则创建新节点
		if !node.hasChild(part) {
			node.addChild(part, newNode())

			if i == 0 {
				// 如果是第一个组件（最左边的子域），则标记为完整的域名规则（叶子节点）
				node.getChild(part).leaf = true
				return true, nil // 插入成功
			}
		}

		// 移动到下一个节点
		node = node.getChild(part)
	}
	return false, nil
}
