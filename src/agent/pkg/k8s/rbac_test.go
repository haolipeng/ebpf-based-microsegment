package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestCheckPermissions_AllAllowed(t *testing.T) {
	// 创建 fake clientset
	fakeClient := fake.NewSimpleClientset()

	// 添加 reactor 来模拟所有权限检查都通过
	fakeClient.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: true,
			},
		}, nil
	})

	client := &Client{
		clientset: fakeClient,
	}

	// 检查权限应该成功
	err := client.CheckPermissions()
	require.NoError(t, err)
}

func TestCheckPermissions_SomeDenied(t *testing.T) {
	// 创建 fake clientset
	fakeClient := fake.NewSimpleClientset()

	// 添加 reactor 来模拟部分权限检查失败
	fakeClient.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(k8stesting.CreateAction)
		ssar := createAction.GetObject().(*authv1.SelfSubjectAccessReview)

		// 允许 pods.list 和 pods.get, 拒绝其他
		allowed := false
		if ssar.Spec.ResourceAttributes != nil {
			if ssar.Spec.ResourceAttributes.Resource == "pods" &&
				(ssar.Spec.ResourceAttributes.Verb == "list" || ssar.Spec.ResourceAttributes.Verb == "get") {
				allowed = true
			}
		}

		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: allowed,
			},
		}, nil
	})

	client := &Client{
		clientset: fakeClient,
	}

	// 检查权限应该失败
	err := client.CheckPermissions()
	require.Error(t, err)

	// 错误信息应该包含缺失的权限
	assert.Contains(t, err.Error(), "Missing required Kubernetes RBAC permissions")
	assert.Contains(t, err.Error(), "watch.pods")  // 应该提到缺少 watch.pods
	assert.Contains(t, err.Error(), "services")    // 应该提到缺少 services 权限

	// 错误信息应该包含 RBAC 配置建议
	assert.Contains(t, err.Error(), "apiVersion: rbac.authorization.k8s.io/v1")
	assert.Contains(t, err.Error(), "kind: ClusterRole")
	assert.Contains(t, err.Error(), "kubectl apply -f rbac.yaml")
}

func TestCheckPermissions_AllDenied(t *testing.T) {
	// 创建 fake clientset
	fakeClient := fake.NewSimpleClientset()

	// 添加 reactor 来模拟所有权限检查都失败
	fakeClient.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: false,
			},
		}, nil
	})

	client := &Client{
		clientset: fakeClient,
	}

	// 检查权限应该失败
	err := client.CheckPermissions()
	require.Error(t, err)

	// 错误信息应该包含所有缺失的权限
	assert.Contains(t, err.Error(), "pods")
	assert.Contains(t, err.Error(), "services")
	assert.Contains(t, err.Error(), "list")
	assert.Contains(t, err.Error(), "watch")
	assert.Contains(t, err.Error(), "get")
}

func TestCheckPermissionsWithNamespace(t *testing.T) {
	// 创建 fake clientset
	fakeClient := fake.NewSimpleClientset()

	// 记录检查的 namespace
	var checkedNamespace string

	// 添加 reactor
	fakeClient.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(k8stesting.CreateAction)
		ssar := createAction.GetObject().(*authv1.SelfSubjectAccessReview)

		// 记录 namespace
		if ssar.Spec.ResourceAttributes != nil {
			checkedNamespace = ssar.Spec.ResourceAttributes.Namespace
		}

		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: true,
			},
		}, nil
	})

	client := &Client{
		clientset: fakeClient,
	}

	// 检查特定 namespace 的权限
	err := client.CheckPermissionsWithNamespace("production")
	require.NoError(t, err)

	// 验证检查的是正确的 namespace
	assert.Equal(t, "production", checkedNamespace)
}

func TestBuildPermissionError(t *testing.T) {
	client := &Client{}

	failedPermissions := []Permission{
		{Resource: "pods", Verb: "list", Group: ""},
		{Resource: "pods", Verb: "watch", Group: ""},
		{Resource: "services", Verb: "get", Group: ""},
	}

	err := client.buildPermissionError(failedPermissions)
	require.Error(t, err)

	errMsg := err.Error()

	// 验证错误信息包含所有必要的部分
	assert.Contains(t, errMsg, "Missing required Kubernetes RBAC permissions")
	assert.Contains(t, errMsg, "list.pods")
	assert.Contains(t, errMsg, "watch.pods")
	assert.Contains(t, errMsg, "get.services")

	// 验证 RBAC 配置建议
	assert.Contains(t, errMsg, "kind: ServiceAccount")
	assert.Contains(t, errMsg, "name: microsegment-agent")
	assert.Contains(t, errMsg, "kind: ClusterRole")
	assert.Contains(t, errMsg, "kind: ClusterRoleBinding")
	assert.Contains(t, errMsg, "resources: [\"pods\"]")
	assert.Contains(t, errMsg, "resources: [\"services\"]")
	assert.Contains(t, errMsg, "verbs:")
	assert.Contains(t, errMsg, "kubectl apply -f rbac.yaml")
}
