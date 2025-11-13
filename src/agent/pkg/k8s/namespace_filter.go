package k8s

import (
	log "github.com/sirupsen/logrus"
)

// NamespaceFilter 用于过滤 Kubernetes Namespace
type NamespaceFilter struct {
	// Include 包含的 namespace 列表 (空表示全部)
	include map[string]bool
	// Exclude 排除的 namespace 列表 (优先级高于 Include)
	exclude map[string]bool
	// includeAll 是否包含所有 namespace (当 Include 为空时)
	includeAll bool
}

// NamespaceFilterConfig Namespace 过滤器配置
type NamespaceFilterConfig struct {
	// Include 包含的 namespace 列表
	Include []string
	// Exclude 排除的 namespace 列表
	Exclude []string
}

// NewNamespaceFilter 创建新的 Namespace 过滤器
func NewNamespaceFilter(cfg *NamespaceFilterConfig) *NamespaceFilter {
	if cfg == nil {
		cfg = &NamespaceFilterConfig{}
	}

	filter := &NamespaceFilter{
		include:    make(map[string]bool),
		exclude:    make(map[string]bool),
		includeAll: len(cfg.Include) == 0,
	}

	// 构建 include map
	for _, ns := range cfg.Include {
		filter.include[ns] = true
	}

	// 构建 exclude map
	for _, ns := range cfg.Exclude {
		filter.exclude[ns] = true
	}

	log.WithFields(log.Fields{
		"include_count": len(filter.include),
		"exclude_count": len(filter.exclude),
		"include_all":   filter.includeAll,
	}).Debug("Namespace filter created")

	return filter
}

// ShouldInclude 判断是否应该包含指定的 namespace
// 过滤规则:
// 1. 如果在 Exclude 列表中,返回 false
// 2. 如果 Include 列表为空 (includeAll=true),返回 true
// 3. 如果在 Include 列表中,返回 true
// 4. 否则返回 false
func (f *NamespaceFilter) ShouldInclude(namespace string) bool {
	// 1. Exclude 优先级最高
	if f.exclude[namespace] {
		log.WithField("namespace", namespace).Trace("Namespace excluded by exclude list")
		return false
	}

	// 2. 如果 Include 为空,则包含所有未被 Exclude 的 namespace
	if f.includeAll {
		log.WithField("namespace", namespace).Trace("Namespace included (include_all)")
		return true
	}

	// 3. 检查是否在 Include 列表中
	if f.include[namespace] {
		log.WithField("namespace", namespace).Trace("Namespace included by include list")
		return true
	}

	// 4. 默认不包含
	log.WithField("namespace", namespace).Trace("Namespace not in include list")
	return false
}

// GetIncludedNamespaces 返回明确指定要包含的 namespace 列表
// 如果 includeAll=true,返回空列表 (表示所有 namespace)
func (f *NamespaceFilter) GetIncludedNamespaces() []string {
	if f.includeAll {
		return []string{} // 空列表表示所有 namespace
	}

	namespaces := make([]string, 0, len(f.include))
	for ns := range f.include {
		namespaces = append(namespaces, ns)
	}
	return namespaces
}

// GetExcludedNamespaces 返回要排除的 namespace 列表
func (f *NamespaceFilter) GetExcludedNamespaces() []string {
	namespaces := make([]string, 0, len(f.exclude))
	for ns := range f.exclude {
		namespaces = append(namespaces, ns)
	}
	return namespaces
}

// IsIncludeAll 返回是否包含所有 namespace (除了 Exclude 列表中的)
func (f *NamespaceFilter) IsIncludeAll() bool {
	return f.includeAll
}

// String 返回过滤器的字符串表示
func (f *NamespaceFilter) String() string {
	if f.includeAll {
		if len(f.exclude) > 0 {
			return "all namespaces except excluded"
		}
		return "all namespaces"
	}

	if len(f.include) > 0 {
		if len(f.exclude) > 0 {
			return "specified namespaces (with exclusions)"
		}
		return "specified namespaces only"
	}

	return "no namespaces"
}
