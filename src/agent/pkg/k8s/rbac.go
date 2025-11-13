package k8s

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Permission 表示需要检查的 Kubernetes 权限
type Permission struct {
	// Resource 资源类型 (例如: "pods", "services")
	Resource string
	// Verb 操作类型 (例如: "list", "watch", "get")
	Verb string
	// Group API 组 (例如: "" 表示 core group, "apps" 等)
	Group string
	// Namespace 命名空间 (空字符串表示集群级别)
	Namespace string
}

// CheckPermissions 检查客户端是否具有所需的 Kubernetes RBAC 权限
// 返回 nil 表示所有权限检查通过,否则返回详细的错误信息和 RBAC 配置建议
func (c *Client) CheckPermissions() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 定义需要检查的权限列表
	requiredPermissions := []Permission{
		// Pod 权限
		{Resource: "pods", Verb: "list", Group: "", Namespace: ""},
		{Resource: "pods", Verb: "watch", Group: "", Namespace: ""},
		{Resource: "pods", Verb: "get", Group: "", Namespace: ""},
		// Service 权限
		{Resource: "services", Verb: "list", Group: "", Namespace: ""},
		{Resource: "services", Verb: "watch", Group: "", Namespace: ""},
		{Resource: "services", Verb: "get", Group: "", Namespace: ""},
	}

	var failedPermissions []Permission

	log.Info("Checking Kubernetes RBAC permissions...")

	// 逐个检查权限
	for _, perm := range requiredPermissions {
		allowed, err := c.checkPermission(ctx, perm)
		if err != nil {
			return fmt.Errorf("failed to check permission %s.%s: %w", perm.Verb, perm.Resource, err)
		}

		if !allowed {
			failedPermissions = append(failedPermissions, perm)
			log.WithFields(log.Fields{
				"resource": perm.Resource,
				"verb":     perm.Verb,
				"group":    perm.Group,
			}).Warn("Missing required permission")
		} else {
			log.WithFields(log.Fields{
				"resource": perm.Resource,
				"verb":     perm.Verb,
				"group":    perm.Group,
			}).Debug("Permission check passed")
		}
	}

	// 如果有失败的权限检查,生成详细的错误信息
	if len(failedPermissions) > 0 {
		return c.buildPermissionError(failedPermissions)
	}

	log.Info("All RBAC permission checks passed")
	return nil
}

// checkPermission 检查单个权限
func (c *Client) checkPermission(ctx context.Context, perm Permission) (bool, error) {
	// 构建 SelfSubjectAccessReview 请求
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: perm.Namespace,
				Verb:      perm.Verb,
				Group:     perm.Group,
				Resource:  perm.Resource,
			},
		},
	}

	// 发送权限检查请求
	result, err := c.clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(
		ctx,
		ssar,
		metav1.CreateOptions{},
	)
	if err != nil {
		return false, fmt.Errorf("SelfSubjectAccessReview failed: %w", err)
	}

	return result.Status.Allowed, nil
}

// buildPermissionError 构建详细的权限错误信息,包含 RBAC 配置建议
func (c *Client) buildPermissionError(failedPermissions []Permission) error {
	var sb strings.Builder

	sb.WriteString("Missing required Kubernetes RBAC permissions:\n\n")

	// 列出失败的权限
	sb.WriteString("Missing permissions:\n")
	for _, perm := range failedPermissions {
		group := perm.Group
		if group == "" {
			group = "core"
		}
		sb.WriteString(fmt.Sprintf("  - %s.%s (%s API group)\n", perm.Verb, perm.Resource, group))
	}

	// 生成 RBAC 配置建议
	sb.WriteString("\n")
	sb.WriteString("To fix this issue, apply the following RBAC configuration:\n")
	sb.WriteString("\n")
	sb.WriteString("---\n")
	sb.WriteString("apiVersion: v1\n")
	sb.WriteString("kind: ServiceAccount\n")
	sb.WriteString("metadata:\n")
	sb.WriteString("  name: microsegment-agent\n")
	sb.WriteString("  namespace: kube-system\n")
	sb.WriteString("---\n")
	sb.WriteString("apiVersion: rbac.authorization.k8s.io/v1\n")
	sb.WriteString("kind: ClusterRole\n")
	sb.WriteString("metadata:\n")
	sb.WriteString("  name: microsegment-agent\n")
	sb.WriteString("rules:\n")

	// 根据失败的权限生成 rules
	resourceVerbs := make(map[string][]string)
	for _, perm := range failedPermissions {
		resourceVerbs[perm.Resource] = append(resourceVerbs[perm.Resource], perm.Verb)
	}

	for resource, verbs := range resourceVerbs {
		sb.WriteString(fmt.Sprintf("  - apiGroups: [\"\"]\n"))
		sb.WriteString(fmt.Sprintf("    resources: [\"%s\"]\n", resource))
		sb.WriteString(fmt.Sprintf("    verbs: ["))
		for i, verb := range verbs {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("\"%s\"", verb))
		}
		sb.WriteString("]\n")
	}

	sb.WriteString("---\n")
	sb.WriteString("apiVersion: rbac.authorization.k8s.io/v1\n")
	sb.WriteString("kind: ClusterRoleBinding\n")
	sb.WriteString("metadata:\n")
	sb.WriteString("  name: microsegment-agent\n")
	sb.WriteString("roleRef:\n")
	sb.WriteString("  apiGroup: rbac.authorization.k8s.io\n")
	sb.WriteString("  kind: ClusterRole\n")
	sb.WriteString("  name: microsegment-agent\n")
	sb.WriteString("subjects:\n")
	sb.WriteString("  - kind: ServiceAccount\n")
	sb.WriteString("    name: microsegment-agent\n")
	sb.WriteString("    namespace: kube-system\n")
	sb.WriteString("\n")
	sb.WriteString("Save this configuration to a file (e.g., rbac.yaml) and apply it with:\n")
	sb.WriteString("  kubectl apply -f rbac.yaml\n")

	return errors.New(sb.String())
}

// CheckPermissionsWithNamespace 检查特定 Namespace 的权限
func (c *Client) CheckPermissionsWithNamespace(namespace string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 定义需要检查的权限列表 (带 Namespace)
	requiredPermissions := []Permission{
		{Resource: "pods", Verb: "list", Group: "", Namespace: namespace},
		{Resource: "pods", Verb: "watch", Group: "", Namespace: namespace},
		{Resource: "pods", Verb: "get", Group: "", Namespace: namespace},
		{Resource: "services", Verb: "list", Group: "", Namespace: namespace},
		{Resource: "services", Verb: "watch", Group: "", Namespace: namespace},
		{Resource: "services", Verb: "get", Group: "", Namespace: namespace},
	}

	var failedPermissions []Permission

	log.WithField("namespace", namespace).Info("Checking Kubernetes RBAC permissions for namespace")

	// 逐个检查权限
	for _, perm := range requiredPermissions {
		allowed, err := c.checkPermission(ctx, perm)
		if err != nil {
			return fmt.Errorf("failed to check permission %s.%s in namespace %s: %w",
				perm.Verb, perm.Resource, namespace, err)
		}

		if !allowed {
			failedPermissions = append(failedPermissions, perm)
		}
	}

	// 如果有失败的权限检查,生成详细的错误信息
	if len(failedPermissions) > 0 {
		return c.buildPermissionError(failedPermissions)
	}

	log.WithField("namespace", namespace).Info("All RBAC permission checks passed for namespace")
	return nil
}
