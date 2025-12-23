// input: test context, expected/actual values
// output: assertion results with detailed error messages
// pos: testutil - custom assertion helpers for test validation

package testutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flowpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/flow"
	policypb "github.com/haolipeng/ebpf-based-microsegment/api/proto/policy"
)

// AssertFlowEventEqual 断言两个 FlowEvent 相等
// 忽略 timestamp 字段的精确匹配（允许误差）
func AssertFlowEventEqual(t *testing.T, expected, actual *flowpb.FlowEvent) {
	t.Helper()

	assert.Equal(t, expected.SrcIp, actual.SrcIp, "Source IP mismatch")
	assert.Equal(t, expected.DstIp, actual.DstIp, "Destination IP mismatch")
	assert.Equal(t, expected.SrcPort, actual.SrcPort, "Source port mismatch")
	assert.Equal(t, expected.DstPort, actual.DstPort, "Destination port mismatch")
	assert.Equal(t, expected.Protocol, actual.Protocol, "Protocol mismatch")
	assert.Equal(t, expected.EventType, actual.EventType, "Event type mismatch")
	assert.Equal(t, expected.Direction, actual.Direction, "Direction mismatch")
	assert.Equal(t, expected.PolicyId, actual.PolicyId, "Policy ID mismatch")
	assert.Equal(t, expected.PolicyAction, actual.PolicyAction, "Policy action mismatch")
	assert.Equal(t, expected.State, actual.State, "State mismatch")
	assert.Equal(t, expected.AgentId, actual.AgentId, "Agent ID mismatch")
	assert.Equal(t, expected.SourceLabels, actual.SourceLabels, "Source labels mismatch")
	assert.Equal(t, expected.DestLabels, actual.DestLabels, "Destination labels mismatch")

	// 允许 timestamp 有 1 秒的误差
	if expected.TimestampNs != 0 {
		diff := int64(actual.TimestampNs) - int64(expected.TimestampNs)
		if diff < 0 {
			diff = -diff
		}
		assert.Less(t, diff, int64(time.Second), "Timestamp difference too large")
	}
}

// AssertPolicyEqual 断言两个 Policy 相等
func AssertPolicyEqual(t *testing.T, expected, actual *policypb.Policy) {
	t.Helper()

	assert.Equal(t, expected.RuleId, actual.RuleId, "Rule ID mismatch")
	assert.Equal(t, expected.SrcIp, actual.SrcIp, "Source IP mismatch")
	assert.Equal(t, expected.DstIp, actual.DstIp, "Destination IP mismatch")
	assert.Equal(t, expected.SrcPort, actual.SrcPort, "Source port mismatch")
	assert.Equal(t, expected.DstPort, actual.DstPort, "Destination port mismatch")
	assert.Equal(t, expected.Protocol, actual.Protocol, "Protocol mismatch")
	assert.Equal(t, expected.Action, actual.Action, "Action mismatch")
	assert.Equal(t, expected.Priority, actual.Priority, "Priority mismatch")
	assert.Equal(t, expected.Description, actual.Description, "Description mismatch")
	assert.Equal(t, expected.SourceLabels, actual.SourceLabels, "Source labels mismatch")
	assert.Equal(t, expected.DestLabels, actual.DestLabels, "Destination labels mismatch")
}

// RequireNoError 要求没有错误（如果有错误则立即失败）
// 这是 testify require.NoError 的语义化封装
func RequireNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	require.NoError(t, err, msgAndArgs...)
}

// AssertEventually 在超时时间内重复检查条件是否满足
// 用于异步操作的测试
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, tick time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Eventually(t, condition, timeout, tick, msgAndArgs...)
}

// AssertInDelta 断言两个数值在允许的误差范围内
func AssertInDelta(t *testing.T, expected, actual interface{}, delta float64, msgAndArgs ...interface{}) {
	t.Helper()
	assert.InDelta(t, expected, actual, delta, msgAndArgs...)
}

// AssertContains 断言字符串包含子字符串
func AssertContains(t *testing.T, s, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Contains(t, s, substr, msgAndArgs...)
}

// AssertNotEmpty 断言值不为空
func AssertNotEmpty(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.NotEmpty(t, object, msgAndArgs...)
}

// AssertGreaterThan 断言 e1 > e2
func AssertGreaterThan(t *testing.T, e1, e2 interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Greater(t, e1, e2, msgAndArgs...)
}

// AssertLessThan 断言 e1 < e2
func AssertLessThan(t *testing.T, e1, e2 interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Less(t, e1, e2, msgAndArgs...)
}

// AssertWithinDuration 断言两个时间在允许的时间范围内
func AssertWithinDuration(t *testing.T, expected, actual time.Time, delta time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	assert.WithinDuration(t, expected, actual, delta, msgAndArgs...)
}

// AssertJSONEqual 断言两个 JSON 字符串等价（忽略格式）
func AssertJSONEqual(t *testing.T, expected, actual string, msgAndArgs ...interface{}) {
	t.Helper()
	assert.JSONEq(t, expected, actual, msgAndArgs...)
}

// AssertSubset 断言 list 包含 subset 中的所有元素
func AssertSubset(t *testing.T, list, subset interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Subset(t, list, subset, msgAndArgs...)
}

// AssertLen 断言对象的长度
func AssertLen(t *testing.T, object interface{}, length int, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Len(t, object, length, msgAndArgs...)
}

// AssertEmpty 断言对象为空
func AssertEmpty(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Empty(t, object, msgAndArgs...)
}

// AssertNil 断言对象为 nil
func AssertNil(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Nil(t, object, msgAndArgs...)
}

// AssertNotNil 断言对象不为 nil
func AssertNotNil(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.NotNil(t, object, msgAndArgs...)
}

// AssertTrue 断言条件为真
func AssertTrue(t *testing.T, value bool, msgAndArgs ...interface{}) {
	t.Helper()
	assert.True(t, value, msgAndArgs...)
}

// AssertFalse 断言条件为假
func AssertFalse(t *testing.T, value bool, msgAndArgs ...interface{}) {
	t.Helper()
	assert.False(t, value, msgAndArgs...)
}
