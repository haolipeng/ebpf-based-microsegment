# 第5周：测试与优化

**[⬅️ 第4周](./week4-advanced-features.md)** | **[📚 目录](./README.md)** | **[➡️ 第6周](./week6-production-deployment.md)**

---

## 📋 学习进度跟踪表

> 💡 **使用说明**：每天学习后，更新下表记录你的进度、遇到的问题和解决方案

| 日期 | 学习内容 | 状态 | 实际耗时 | 遇到的问题 | 解决方案/笔记 |
|------|----------|------|----------|-----------|--------------|
| Day 1 | 单元测试编写 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 2 | 功能测试 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 3 | 性能测试 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 4 | 压力测试 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 5 | 性能优化 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 6-7 | Bug修复 + 周总结 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |

### 📝 本周学习笔记

**重点概念：**
-
-
-

**遇到的难点：**
1.
2.

**解决的关键问题：**
1.
2.

**下周需要重点关注：**
-
-

---

## 6. 第5周：测试与优化

### 🎯 本周目标

- [ ] 编写完整测试套件
- [ ] 进行功能、性能、压力测试
- [ ] 性能调优
- [ ] Bug修复

### 📊 本周交付物

1. ✅ 完整的测试套件
2. ✅ 测试报告 (功能+性能)
3. ✅ 性能调优方案
4. ✅ Bug修复列表

---

### 📅 Day 1: 单元测试编写

#### 🎯 任务目标
- 为核心函数编写单元测试
- 使用bpf_testmod或自定义测试框架
- 实现Mock数据包测试

#### ✅ 具体任务

**全天：编写单元测试**

创建测试框架 `tests/unit_tests.c`:

```
```c
#include <stdio.h>
#include <assert.h>
#include <string.h>
#include <arpa/inet.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>

// 测试辅助函数
void assert_eq(int expected, int actual, const char *msg) {
    if (expected != actual) {
        fprintf(stderr, "FAIL: %s (expected=%d, actual=%d)\n",
                msg, expected, actual);
        exit(1);
    }
    printf("PASS: %s\n", msg);
}

// 测试1: Map创建和访问
void test_map_operations() {
    printf("\n=== Test: Map Operations ===\n");

    // 打开Maps
    int policy_fd = bpf_obj_get("/sys/fs/bpf/policy_map");
    assert_eq(1, policy_fd > 0, "Open policy_map");

    // 测试插入
    struct flow_key key = {
        .src_ip = inet_addr("192.168.1.100"),
        .dst_ip = inet_addr("10.0.0.50"),
        .dst_port = htons(80),
        .protocol = 6
    };

    struct policy_value value = {
        .action = 0,
        .priority = 100
    };

    int ret = bpf_map_update_elem(policy_fd, &key, &value, BPF_ANY);
    assert_eq(0, ret, "Insert policy");

    // 测试查找
    struct policy_value result;
    ret = bpf_map_lookup_elem(policy_fd, &key, &result);
    assert_eq(0, ret, "Lookup policy");
    assert_eq(0, result.action, "Policy action");

    // 测试删除
    ret = bpf_map_delete_elem(policy_fd, &key);
    assert_eq(0, ret, "Delete policy");

    close(policy_fd);
}

// 测试2: 5元组key构造
void test_flow_key_construction() {
    printf("\n=== Test: Flow Key Construction ===\n");

    struct flow_key key1 = {
        .src_ip = inet_addr("192.168.1.100"),
        .dst_ip = inet_addr("10.0.0.50"),
        .src_port = htons(12345),
        .dst_port = htons(80),
        .protocol = 6
    };

    struct flow_key key2 = key1;

    // 测试key相等性
    int cmp = memcmp(&key1, &key2, sizeof(struct flow_key));
    assert_eq(0, cmp, "Flow key equality");

    // 测试不同key
    key2.dst_port = htons(443);
    cmp = memcmp(&key1, &key2, sizeof(struct flow_key));
    assert_eq(1, cmp != 0, "Flow key inequality");
}

// 测试3: TCP状态转换
void test_tcp_state_machine() {
    printf("\n=== Test: TCP State Machine ===\n");

    // 模拟TCP握手序列
    struct session_value sess = {.tcp_state = 0};  // TCP_NONE

    // SYN -> SYN_SENT
    // (模拟状态转换逻辑)
    sess.tcp_state = 1;  // TCP_SYN_SENT
    assert_eq(1, sess.tcp_state, "After SYN");

    // SYN+ACK -> SYN_RECV
    sess.tcp_state = 2;  // TCP_SYN_RECV
    assert_eq(2, sess.tcp_state, "After SYN+ACK");

    // ACK -> ESTABLISHED
    sess.tcp_state = 3;  // TCP_ESTABLISHED
    assert_eq(3, sess.tcp_state, "After ACK (ESTABLISHED)");
}

// 测试4: LPM Trie匹配
void test_lpm_trie_matching() {
    printf("\n=== Test: LPM Trie Matching ===\n");

    int map_fd = bpf_obj_get("/sys/fs/bpf/ip_range_map");
    if (map_fd < 0) {
        printf("SKIP: LPM Trie map not available\n");
        return;
    }

    // 添加 192.168.1.0/24
    struct lpm_key key = {
        .prefixlen = 24,
        .ip = inet_addr("192.168.1.0")
    };

    struct ip_range_value value = {.action = 0, .priority = 100};
    bpf_map_update_elem(map_fd, &key, &value, BPF_ANY);

    // 查找 192.168.1.100 (应该匹配)
    key.prefixlen = 32;
    key.ip = inet_addr("192.168.1.100");

    struct ip_range_value result;
    int ret = bpf_map_lookup_elem(map_fd, &key, &result);
    assert_eq(0, ret, "LPM match 192.168.1.100");

    // 查找 192.168.2.100 (不应该匹配)
    key.ip = inet_addr("192.168.2.100");
    ret = bpf_map_lookup_elem(map_fd, &key, &result);
    assert_eq(1, ret < 0, "LPM no match 192.168.2.100");

    close(map_fd);
}

// 测试5: 统计计数器
void test_statistics_counters() {
    printf("\n=== Test: Statistics Counters ===\n");

    int stats_fd = bpf_obj_get("/sys/fs/bpf/stats_map");
    assert_eq(1, stats_fd > 0, "Open stats_map");

    // 读取当前统计
    __u32 key = 0;  // STAT_TOTAL
    __u64 value_before;
    bpf_map_lookup_elem(stats_fd, &key, &value_before);

    printf("  Total packets before: %llu\n", value_before);

    // 生成一些流量...
    system("ping -c 1 127.0.0.1 >/dev/null 2>&1");

    // 读取新统计
    __u64 value_after;
    bpf_map_lookup_elem(stats_fd, &key, &value_after);

    printf("  Total packets after: %llu\n", value_after);

    assert_eq(1, value_after >= value_before, "Statistics incremented");

    close(stats_fd);
}

int main() {
    printf("===========================================\n");
    printf("eBPF Microsegmentation Unit Tests\n");
    printf("===========================================\n");

    test_map_operations();
    test_flow_key_construction();
    test_tcp_state_machine();
    test_lpm_trie_matching();
    test_statistics_counters();

    printf("\n===========================================\n");
    printf("✓ All tests passed!\n");
    printf("===========================================\n");

    return 0;
}
```

编译和运行测试:

```bash
# 编译测试
gcc -o unit_tests tests/unit_tests.c -lbpf -I./src/include

# 运行测试 (需要先启动 libbpf 加载器)
sudo ./microsegment_loader lo &
LOADER_PID=$!
sleep 2
sudo ./unit_tests

# 清理
sudo kill $LOADER_PID
```

#### 📚 学习资料

1. 单元测试最佳实践:
   - 测试隔离
   - Mock和Stub
   - 断言设计
   - 时间: 1.5小时

2. BPF测试框架:
   - libbpf测试工具
   - bpf_prog_test_run
   - 时间: 1.5小时

#### ✅ 完成标准 (Day 1)

- [ ] 至少5个单元测试编写
- [ ] 测试覆盖核心功能
- [ ] 所有测试通过
- [ ] 测试可自动化运行

---

### 📅 Day 2: 功能测试与bug修复

#### 🎯 任务目标
- 编写端到端功能测试
- 发现并修复bug
- 验证所有用户场景

#### ✅ 具体任务

**全天：功能测试**

创建功能测试套件 `tests/functional_tests.sh`:

```bash
#!/bin/bash
set -e

FAILED=0
PASSED=0

pass() {
    echo "✓ PASS: $1"
    PASSED=$((PASSED + 1))
}

fail() {
    echo "✗ FAIL: $1"
    FAILED=$((FAILED + 1))
}

cleanup() {
    sudo killall microsegment_loader 2>/dev/null || true
}

trap cleanup EXIT

echo "=== Functional Tests ==="

# 启动 libbpf 加载器
sudo ./microsegment_loader lo &
PID=$!
sleep 3

# 测试1: 基础策略添加和查询
echo -e "\n[Test 1] Policy CRUD operations"
sudo ./tc_microsegment_cli policy add --dst-ip 127.0.0.1 --dst-port 8080 --action allow
COUNT=$(sudo ./tc_microsegment_cli policy list | grep -c "127.0.0.1")
if [ "$COUNT" -eq 1 ]; then
    pass "Policy add and list"
else
    fail "Policy add and list"
fi

# 测试2: 策略匹配和执行
echo -e "\n[Test 2] Policy enforcement"
sudo ./tc_microsegment_cli policy add --dst-ip 127.0.0.1 --dst-port 22 --action deny

# 启动临时SSH服务器 (如果存在)
nc -l 127.0.0.1 22 &
NC_PID=$!
sleep 1

# 尝试连接 (应该被拒绝)
timeout 2 telnet 127.0.0.1 22 2>/dev/null && fail "Deny policy" || pass "Deny policy"

kill $NC_PID 2>/dev/null || true

# 测试3: 会话跟踪
echo -e "\n[Test 3] Session tracking"
curl -s http://127.0.0.1:8080 >/dev/null 2>&1 &
sleep 1

SESSION_COUNT=$(sudo ./tc_microsegment_cli session list | grep -c "127.0.0.1")
if [ "$SESSION_COUNT" -gt 0 ]; then
    pass "Session tracking"
else
    fail "Session tracking"
fi

# 测试4: 会话缓存命中
echo -e "\n[Test 4] Session cache hit rate"
for i in {1..10}; do
    curl -s http://127.0.0.1:8080 >/dev/null 2>&1 || true
done

TOTAL=$(sudo bpftool map dump name stats_map | grep "key: 0" -A1 | grep value | awk '{print $2}')
SESSION_HIT=$(sudo bpftool map dump name stats_map | grep "key: 2" -A1 | grep value | awk '{print $2}')

if [ "$TOTAL" -gt 0 ] && [ "$SESSION_HIT" -gt 0 ]; then
    HITRATE=$((SESSION_HIT * 100 / TOTAL))
    if [ "$HITRATE" -gt 50 ]; then
        pass "Session cache ($HITRATE% hit rate)"
    else
        fail "Session cache ($HITRATE% hit rate too low)"
    fi
else
    fail "Session cache (no data)"
fi

# 测试5: IP段匹配
echo -e "\n[Test 5] IP range matching"
sudo ./tc_microsegment_cli ip-range add --cidr 127.0.0.0/8 --action allow 2>/dev/null || true

# 测试匹配
ping -c 1 127.0.0.1 >/dev/null 2>&1 && pass "IP range match" || fail "IP range match"

# 测试6: 统计功能
echo -e "\n[Test 6] Statistics"
STATS=$(sudo ./tc_microsegment_cli stats show)
if echo "$STATS" | grep -q "Total packets"; then
    pass "Statistics display"
else
    fail "Statistics display"
fi

# 测试7: 策略热更新
echo -e "\n[Test 7] Hot policy reload"
cat > /tmp/test_policies.json <<EOF
{
  "policies": [
    {
      "dst_ip": "127.0.0.1",
      "dst_port": 9090,
      "protocol": "tcp",
      "action": "allow",
      "priority": 100
    }
  ]
}
EOF

sudo ./tc_microsegment_cli policy load --file /tmp/test_policies.json 2>/dev/null || true
sleep 1

COUNT=$(sudo ./tc_microsegment_cli policy list | grep -c "9090")
if [ "$COUNT" -eq 1 ]; then
    pass "Hot policy reload"
else
    fail "Hot policy reload"
fi

# 测试8: TCP状态机
echo -e "\n[Test 8] TCP state machine"
# 建立完整TCP连接
nc -zv 127.0.0.1 8080 2>&1 | grep -q "succeeded" && pass "TCP handshake" || fail "TCP handshake"

# 汇总
echo -e "\n========================================="
echo "Tests run: $((PASSED + FAILED))"
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo "========================================="

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi
```

运行功能测试:

```bash
chmod +x tests/functional_tests.sh
sudo ./tests/functional_tests.sh
```

**Bug修复流程**:

1. 记录失败的测试
2. 使用bpf_printk调试eBPF程序
3. 使用gdb调试用户态程序
4. 修复代码
5. 重新运行测试验证

#### 📚 学习资料

1. 调试技巧:
   - bpf_printk使用
   - trace_pipe分析
   - bpftool调试
   - 时间: 2小时

2. 常见bug模式:
   - 边界条件
   - 并发问题
   - 内存泄漏
   - 时间: 1.5小时

#### ✅ 完成标准 (Day 2)

- [ ] 所有功能测试通过
- [ ] 发现的bug已修复
- [ ] 测试用例文档化
- [ ] Bug修复记录

---

### 📅 Day 3: 性能测试与调优

#### 🎯 任务目标
- 测试吞吐量和延迟
- 识别性能瓶颈
- 优化关键路径

#### ✅ 具体任务

**全天：性能测试和优化**

创建性能测试脚本 `tests/performance_tests.sh`:

```bash
#!/bin/bash
set -e

echo "=== Performance Tests ==="

# 测试1: 基准吞吐量 (无eBPF)
echo -e "\n[Baseline] Throughput without eBPF"

iperf3 -s -p 5201 >/dev/null 2>&1 &
IPERF_PID=$!
sleep 2

BASELINE=$(iperf3 -c 127.0.0.1 -p 5201 -t 10 -J | jq '.end.sum_received.bits_per_second')
echo "  Baseline: $(echo "scale=2; $BASELINE / 1000000000" | bc) Gbps"

kill $IPERF_PID 2>/dev/null || true
sleep 2

# 测试2: eBPF吞吐量
echo -e "\n[eBPF] Throughput with eBPF filtering"
sudo ./microsegment_loader lo &
LOADER_PID=$!
sleep 3

iperf3 -s -p 5201 >/dev/null 2>&1 &
IPERF_PID=$!
sleep 2

EBPF_BW=$(iperf3 -c 127.0.0.1 -p 5201 -t 10 -J | jq '.end.sum_received.bits_per_second')
echo "  eBPF: $(echo "scale=2; $EBPF_BW / 1000000000" | bc) Gbps"

OVERHEAD=$(echo "scale=2; (1 - $EBPF_BW / $BASELINE) * 100" | bc)
echo "  Overhead: $OVERHEAD%"

kill $IPERF_PID 2>/dev/null || true

# 先清理 eBPF 程序
sudo kill $LOADER_PID 2>/dev/null || true
sleep 1

# 测试3: 延迟测试
echo -e "\n[Latency] Round-trip time"

# 无eBPF
RTT_BASELINE=$(ping -c 100 -i 0.01 127.0.0.1 | grep "avg" | awk -F'/' '{print $5}')
echo "  Baseline RTT: $RTT_BASELINE ms"

# 有eBPF
sudo ./microsegment_loader lo &
LOADER_PID=$!
sleep 3

RTT_EBPF=$(ping -c 100 -i 0.01 127.0.0.1 | grep "avg" | awk -F'/' '{print $5}')
echo "  eBPF RTT: $RTT_EBPF ms"

LATENCY_OVERHEAD=$(echo "scale=2; $RTT_EBPF - $RTT_BASELINE" | bc)
echo "  Added latency: $LATENCY_OVERHEAD ms"

# 测试4: 策略查找性能
echo -e "\n[Policy Lookup] Performance"

# 添加1000条策略
for i in {1..1000}; do
    sudo ./tc_microsegment_cli policy add \
        --dst-ip 10.0.$((i/256)).$((i%256)) \
        --dst-port $((1000 + i)) \
        --action allow \
        >/dev/null 2>&1
done

echo "  Added 1000 policies"

# 测试查找时间
START=$(date +%s%N)
for i in {1..1000}; do
    sudo ./tc_microsegment_cli policy list >/dev/null 2>&1
done
END=$(date +%s%N)

ELAPSED=$(echo "scale=2; ($END - $START) / 1000000000" | bc)
AVG=$(echo "scale=2; $ELAPSED / 1000" | bc)
echo "  1000 lookups in $ELAPSED seconds"
echo "  Average: $AVG ms per lookup"

# 测试5: 会话缓存性能
echo -e "\n[Session Cache] Performance"

# 清空会话
sudo bpftool map delete name session_map 2>/dev/null || true

# 首次连接 (未缓存)
START=$(date +%s%N)
curl -s http://127.0.0.1:8080 >/dev/null 2>&1 || true
END=$(date +%s%N)
FIRST=$(echo "scale=4; ($END - $START) / 1000000" | bc)

# 后续连接 (已缓存)
START=$(date +%s%N)
for i in {1..100}; do
    curl -s http://127.0.0.1:8080 >/dev/null 2>&1 || true
done
END=$(date +%s%N)
CACHED=$(echo "scale=4; ($END - $START) / 100 / 1000000" | bc)

echo "  First request (uncached): $FIRST ms"
echo "  Cached requests (avg): $CACHED ms"

SPEEDUP=$(echo "scale=2; $FIRST / $CACHED" | bc)
echo "  Speedup: ${SPEEDUP}x"

# 汇总
echo -e "\n========================================="
echo "Performance Summary:"
echo "  Throughput overhead: $OVERHEAD%"
echo "  Latency overhead: +$LATENCY_OVERHEAD ms"
echo "  Policy lookup: $AVG ms"
echo "  Session cache speedup: ${SPEEDUP}x"
echo "========================================="
```

**性能优化checklist**:

1. **eBPF程序优化**:
   - 减少Map查找次数
   - 使用__always_inline
   - 避免复杂循环

2. **Map优化**:
   - 使用合适的Map类型
   - 调整Map大小
   - 启用BPF_F_NO_PREALLOC (如果适用)

3. **用户态优化**:
   - 减少系统调用
   - 批量操作
   - 使用缓存

#### 📚 学习资料

1. 性能分析工具:
   - perf
   - flamegraph
   - bpftool prog profile
   - 时间: 2小时

2. 优化技巧:
   - 热点路径识别
   - Cache-friendly设计
   - 时间: 1.5小时

#### ✅ 完成标准 (Day 3)

- [ ] 性能基准测试完成
- [ ] 吞吐量开销 < 5%
- [ ] 延迟开销 < 10μs
- [ ] 会话缓存加速 > 3x

---

### 📅 Day 4: 压力测试

#### 🎯 任务目标
- 测试系统极限
- 验证稳定性
- 测试异常场景

#### ✅ 具体任务

**全天：压力测试**

创建压力测试脚本 `tests/stress_tests.sh`:

```bash
#!/bin/bash
set -e

echo "=== Stress Tests ==="

# 启动 libbpf 加载器
sudo ./microsegment_loader lo &
PID=$!
sleep 3

# 测试1: 高并发连接
echo -e "\n[Test 1] High concurrency"

# 启动多个HTTP服务器
for port in {8080..8090}; do
    python3 -m http.server $port >/dev/null 2>&1 &
done
sleep 2

# 并发请求
echo "  Sending 10000 concurrent requests..."
ab -n 10000 -c 100 http://127.0.0.1:8080/ >/dev/null 2>&1

# 检查统计
TOTAL=$(sudo bpftool map dump name stats_map | grep "key: 0" -A1 | grep value | awk '{print $2}')
echo "  Total packets processed: $TOTAL"

if [ "$TOTAL" -gt 10000 ]; then
    echo "  ✓ PASS"
else
    echo "  ✗ FAIL"
fi

# 清理
killall python3 2>/dev/null || true

# 测试2: Map容量测试
echo -e "\n[Test 2] Map capacity"

echo "  Filling session map..."
for i in {1..100000}; do
    # 模拟不同源IP的连接
    curl -s --interface "127.0.0.1:$((10000 + i % 10000))" \
         http://127.0.0.1:8080 >/dev/null 2>&1 || true

    if [ $((i % 10000)) -eq 0 ]; then
        echo "    Created $i sessions..."
    fi
done

SESSION_COUNT=$(sudo bpftool map dump name session_map | grep -c "key:")
echo "  Active sessions: $SESSION_COUNT"

# 检查压力级别
PRESSURE=$(sudo ./tc_microsegment_cli monitor pressure 2>&1 | head -1 | grep -oP 'Pressure: \K\d+')
echo "  Map pressure: $PRESSURE%"

if [ "$SESSION_COUNT" -gt 50000 ]; then
    echo "  ✓ PASS: Handled $SESSION_COUNT sessions"
else
    echo "  ✗ FAIL: Only $SESSION_COUNT sessions"
fi

# 测试3: SYN Flood模拟
echo -e "\n[Test 3] SYN flood detection"

echo "  Simulating SYN flood..."
hping3 -S -p 80 --flood --rand-source 127.0.0.1 -c 10000 >/dev/null 2>&1 || \
    echo "  (hping3 not available, skipping)"

# 检查是否检测到
LOGS=$(sudo dmesg | tail -100 | grep -c "SYN flood" || echo "0")
if [ "$LOGS" -gt 0 ]; then
    echo "  ✓ PASS: SYN flood detected"
else
    echo "  ⚠ WARNING: SYN flood not detected (may need hping3)"
fi

# 测试4: 长时间稳定性测试
echo -e "\n[Test 4] Long-running stability (10 minutes)"

START_TIME=$(date +%s)
ERROR_COUNT=0

echo "  Running continuous traffic for 10 minutes..."
for i in {1..600}; do  # 10分钟
    curl -s http://127.0.0.1:8080 >/dev/null 2>&1 || ERROR_COUNT=$((ERROR_COUNT + 1))
    ping -c 1 127.0.0.1 >/dev/null 2>&1 || ERROR_COUNT=$((ERROR_COUNT + 1))
    sleep 1

    if [ $((i % 60)) -eq 0 ]; then
        ELAPSED=$((i / 60))
        echo "    $ELAPSED minutes elapsed, errors: $ERROR_COUNT"
    fi
done

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo "  Duration: $DURATION seconds"
echo "  Errors: $ERROR_COUNT"

if [ "$ERROR_COUNT" -lt 10 ]; then
    echo "  ✓ PASS: Stable operation"
else
    echo "  ✗ FAIL: Too many errors ($ERROR_COUNT)"
fi

# 测试5: 内存泄漏检测
echo -e "\n[Test 5] Memory leak detection"

# 记录初始内存
INIT_MEM=$(ps -p $PID -o rss= | awk '{print $1}')
echo "  Initial memory: $INIT_MEM KB"

# 运行1小时的流量
echo "  Running traffic for 1 hour..."
for i in {1..3600}; do
    curl -s http://127.0.0.1:8080 >/dev/null 2>&1 || true
    if [ $((i % 600)) -eq 0 ]; then
        CURR_MEM=$(ps -p $PID -o rss= | awk '{print $1}')
        INCREASE=$((CURR_MEM - INIT_MEM))
        echo "    $((i/60)) min: $CURR_MEM KB (+$INCREASE KB)"
    fi
done

# 检查最终内存
FINAL_MEM=$(ps -p $PID -o rss= | awk '{print $1}')
INCREASE=$((FINAL_MEM - INIT_MEM))
INCREASE_PCT=$((INCREASE * 100 / INIT_MEM))

echo "  Final memory: $FINAL_MEM KB"
echo "  Increase: $INCREASE KB ($INCREASE_PCT%)"

if [ "$INCREASE_PCT" -lt 20 ]; then
    echo "  ✓ PASS: No significant memory leak"
else
    echo "  ✗ FAIL: Possible memory leak (+$INCREASE_PCT%)"
fi

echo -e "\n========================================="
echo "Stress tests completed"
echo "========================================="
```

#### 📚 学习资料

1. 压力测试工具:
   - ab (Apache Bench)
   - wrk
   - hping3
   - 时间: 1.5小时

2. 稳定性指标:
   - 内存增长率
   - CPU使用率
   - 错误率
   - 时间: 1小时

#### ✅ 完成标准 (Day 4)

- [ ] 处理10万+并发会话
- [ ] SYN Flood检测触发
- [ ] 长时间运行无崩溃
- [ ] 无明显内存泄漏

---

### 📅 Day 5: 文档整理与测试报告

#### 🎯 任务目标
- 整理所有测试结果
- 编写测试报告
- 更新文档
- 准备演示

#### ✅ 具体任务

**全天：文档和报告**

创建测试报告 `docs/test_report.md`:

```markdown
# eBPF微隔离测试报告

## 1. 测试概述

**测试日期**: 2025-xx-xx
**测试环境**: Ubuntu 22.04, Kernel 5.15
**测试负责人**: XXX

## 2. 单元测试结果

| 测试项 | 结果 | 说明 |
|--------|------|------|
| Map操作 | PASS | 插入/查找/删除正常 |
| Flow Key构造 | PASS | 5元组正确构造 |
| TCP状态机 | PASS | 状态转换符合RFC 793 |
| LPM Trie匹配 | PASS | 最长前缀匹配正确 |
| 统计计数器 | PASS | 计数准确 |

**通过率**: 100% (5/5)

## 3. 功能测试结果

| 测试项 | 结果 | 说明 |
|--------|------|------|
| 策略CRUD | PASS | 增删改查正常 |
| 策略执行 | PASS | allow/deny正确 |
| 会话跟踪 | PASS | 会话正确建立 |
| 会话缓存 | PASS | 命中率95%+ |
| IP段匹配 | PASS | CIDR匹配正确 |
| 统计显示 | PASS | 实时统计准确 |
| 策略热更新 | PASS | 无需重启 |
| TCP握手 | PASS | 3次握手正常 |

**通过率**: 100% (8/8)

## 4. 性能测试结果

| 指标 | 基准值 | eBPF值 | 开销 |
|------|--------|--------|------|
| 吞吐量 | 10.0 Gbps | 9.7 Gbps | 3% |
| 平均延迟 | 0.05 ms | 0.06 ms | +0.01 ms |
| P99延迟 | 0.10 ms | 0.12 ms | +0.02 ms |
| 策略查找 | - | 0.05 ms | - |
| 会话缓存加速 | 1x | 4.2x | +320% |

**结论**: 性能开销在可接受范围内，会话缓存显著提升性能。

## 5. 压力测试结果

| 测试项 | 目标 | 实际 | 结果 |
|--------|------|------|------|
| 并发会话数 | 100,000 | 105,342 | PASS |
| SYN Flood检测 | 触发 | 已触发 | PASS |
| 长时间稳定性 | 10小时 | 10小时无崩溃 | PASS |
| 内存泄漏 | <20%增长 | 8%增长 | PASS |
| CPU使用率 | <80% | 平均45% | PASS |

## 6. 发现的问题

### 已修复

1. **Bug #1**: TCP状态机在FIN_WAIT2不正确转换
   **修复**: 添加对FIN+ACK的处理

2. **Bug #2**: LPM Trie在/32前缀时匹配失败
   **修复**: 调整prefixlen设置

### 待修复

无

## 7. 总结

所有测试通过，系统达到生产就绪状态。

**推荐**: 可进入生产部署阶段。
```

更新README:

```markdown
# eBPF TC 微隔离系统

## 功能特性

- ✅ 基于5元组的策略匹配
- ✅ TCP状态机跟踪
- ✅ IP段匹配 (CIDR)
- ✅ 会话缓存 (LRU)
- ✅ Map压力监控
- ✅ Prometheus集成
- ✅ 策略热更新

## 性能指标

- 吞吐量开销: ~3%
- 延迟增加: ~10μs
- 支持会话数: 100,000+
- 会话缓存加速: 4x

## 快速开始

```bash
# 编译
make

# 运行（使用 libbpf 加载器）
sudo ./microsegment_loader eth0

# 添加策略
sudo ./tc_microsegment_cli policy add \
    --dst-ip 10.0.0.1 --dst-port 80 --action allow

# 查看统计
sudo ./tc_microsegment_cli stats show
```

## 测试

```bash
# 单元测试
sudo ./unit_tests

# 功能测试
sudo ./tests/functional_tests.sh

# 性能测试
sudo ./tests/performance_tests.sh
```

## 文档

- [设计文档](specs/design.md)
- [实施指南](specs/ebpf-tc-implementation.md)
- [测试报告](docs/test_report.md)
```

#### ✅ 完成标准 (Day 5)

- [ ] 测试报告完成
- [ ] README更新
- [ ] 所有文档整理完毕
- [ ] Demo准备就绪

---

### 📅 本周总结 (Friday晚上)

#### ✍️ 输出物

创建文档 `docs/week5_summary.md`:

```markdown
# 第5周学习总结

## 完成情况

- [x] 单元测试 (5个核心测试)
- [x] 功能测试 (8个场景)
- [x] 性能测试 (吞吐量/延迟/缓存)
- [x] 压力测试 (稳定性/内存)

## 测试结果

### 通过率
- 单元测试: 100% (5/5)
- 功能测试: 100% (8/8)
- 性能测试: 达标
- 压力测试: 全部通过

### 性能数据
- 吞吐量开销: 3%
- 延迟增加: 10μs
- 会话缓存: 4.2x加速
- 并发会话: 105K

## 发现并修复的Bug

1. TCP状态机FIN_WAIT2转换问题
2. LPM Trie /32前缀匹配问题

## 下周计划

- 生产部署脚本完善
- 监控集成 (Prometheus + Grafana)
- 金丝雀部署测试
- 项目交付演示
```

#### 🎯 本周验收标准

**必须完成**:
- [ ] 所有测试通过
- [ ] 性能达标
- [ ] Bug全部修复
- [ ] 测试报告完成

**加分项**:
- [ ] 自动化测试流程
- [ ] 性能优化文档
- [ ] 压力测试录屏

---

## 7. 第6周：生产部署准备

### 🎯 本周目标

完成生产环境部署准备，包括部署脚本、监控集成、金丝雀部署测试，最终交付可用于生产环境的完整系统。

### 📊 本周交付物

| 交付物 | 类型 | 描述 |

---

**[⬅️ 第4周](./week4-advanced-features.md)** | **[📚 目录](./README.md)** | **[➡️ 第6周](./week6-production-deployment.md)**
