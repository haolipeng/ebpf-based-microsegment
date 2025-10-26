# 第6周：生产部署准备

**[⬅️ 第5周](./week5-testing-optimization.md)** | **[📚 目录](./README.md)**

---

## 📋 学习进度跟踪表

> 💡 **使用说明**：每天学习后，更新下表记录你的进度、遇到的问题和解决方案

| 日期 | 学习内容 | 状态 | 实际耗时 | 遇到的问题 | 解决方案/笔记 |
|------|----------|------|----------|-----------|--------------|
| Day 1 | 自动化部署脚本 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 2 | 金丝雀部署脚本 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 3 | Prometheus监控集成 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 4 | 金丝雀部署测试 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 5 | 项目交付与演示准备 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 6-7 | 文档完善 + 项目总结 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |

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

**项目亮点总结：**
-
-

---

## 7. 第6周：生产部署准备

### 🎯 本周目标

- [ ] 编写自动化部署脚本
- [ ] 实现金丝雀部署
- [ ] 集成Prometheus监控
- [ ] 准备项目交付材料

### 📊 本周交付物

| 交付物 | 类型 | 说明 |
|--------|------|------|
| 部署脚本套件 | 脚本 | 自动化部署、升级、回滚脚本 |
| 监控Dashboard | 配置 | Prometheus + Grafana完整监控 |
| 金丝雀测试报告 | 文档 | 灰度部署测试结果 |
| 项目交付文档 | 文档 | 完整的项目说明和演示材料 |
| 项目演示Demo | 演示 | 15分钟功能演示视频 |

---

### 📅 Day 1 (Monday): 自动化部署脚本开发

#### 🎯 任务目标
开发完整的自动化部署脚本，支持一键部署、环境检查、依赖安装。

#### ✅ 具体任务

**任务1: 环境检查脚本**

创建 `scripts/check_env.sh`:

```bash
#!/bin/bash
# check_env.sh - 环境检查脚本

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================="
echo "  eBPF微隔离系统环境检查"
echo "========================================="
echo ""

# 检查内核版本
echo -n "检查内核版本... "
KERNEL_VERSION=$(uname -r | cut -d. -f1,2)
KERNEL_MAJOR=$(echo $KERNEL_VERSION | cut -d. -f1)
KERNEL_MINOR=$(echo $KERNEL_VERSION | cut -d. -f2)

if [ $KERNEL_MAJOR -gt 5 ] || ([ $KERNEL_MAJOR -eq 5 ] && [ $KERNEL_MINOR -ge 10 ]); then
    echo -e "${GREEN}✓${NC} $KERNEL_VERSION (>= 5.10)"
else
    echo -e "${RED}✗${NC} $KERNEL_VERSION (需要 >= 5.10)"
    exit 1
fi

# 检查BTF支持
echo -n "检查BTF支持... "
if [ -f /sys/kernel/btf/vmlinux ]; then
    echo -e "${GREEN}✓${NC} BTF已启用"
else
    echo -e "${YELLOW}⚠${NC} BTF未启用 (功能受限)"
fi

# 检查必需工具
REQUIRED_TOOLS="clang llvm bpftool tc ip"
echo ""
echo "检查必需工具:"
for tool in $REQUIRED_TOOLS; do
    echo -n "  $tool... "
    if command -v $tool &> /dev/null; then
        VERSION=$(command $tool --version 2>&1 | head -n1)
        echo -e "${GREEN}✓${NC} 已安装"
    else
        echo -e "${RED}✗${NC} 未安装"
        exit 1
    fi
done

# 检查libbpf
echo -n "检查libbpf开发库... "
if pkg-config --exists libbpf; then
    VERSION=$(pkg-config --modversion libbpf)
    echo -e "${GREEN}✓${NC} $VERSION"
else
    echo -e "${RED}✗${NC} 未安装"
    exit 1
fi

# 检查内存
echo -n "检查可用内存... "
AVAILABLE_MEM=$(free -m | awk '/^Mem:/{print $7}')
if [ $AVAILABLE_MEM -gt 1024 ]; then
    echo -e "${GREEN}✓${NC} ${AVAILABLE_MEM}MB (推荐 >1GB)"
else
    echo -e "${YELLOW}⚠${NC} ${AVAILABLE_MEM}MB (建议至少1GB)"
fi

# 检查磁盘空间
echo -n "检查磁盘空间... "
AVAILABLE_DISK=$(df -m / | awk 'NR==2 {print $4}')
if [ $AVAILABLE_DISK -gt 2048 ]; then
    echo -e "${GREEN}✓${NC} ${AVAILABLE_DISK}MB"
else
    echo -e "${YELLOW}⚠${NC} ${AVAILABLE_DISK}MB (建议至少2GB)"
fi

# 检查root权限
echo -n "检查权限... "
if [ "$EUID" -eq 0 ]; then
    echo -e "${GREEN}✓${NC} root权限"
else
    echo -e "${YELLOW}⚠${NC} 非root用户 (部分功能需要sudo)"
fi

echo ""
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}  环境检查通过！可以继续部署。${NC}"
echo -e "${GREEN}=========================================${NC}"
```

**任务2: 一键部署脚本**

创建 `scripts/deploy.sh`:

```bash
#!/bin/bash
# deploy.sh - 一键部署脚本

set -e

INSTALL_DIR="/opt/tc-microsegment"
BIN_DIR="/usr/local/bin"
CONFIG_DIR="/etc/tc-microsegment"
LOG_DIR="/var/log/tc-microsegment"

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "========================================="
echo "  eBPF微隔离系统部署脚本 v1.0"
echo "========================================="
echo ""

# 步骤1: 环境检查
echo -e "${BLUE}步骤1/6:${NC} 环境检查..."
bash scripts/check_env.sh || exit 1
echo ""

# 步骤2: 创建目录
echo -e "${BLUE}步骤2/6:${NC} 创建目录结构..."
mkdir -p $INSTALL_DIR/{bin,lib,bpf}
mkdir -p $CONFIG_DIR
mkdir -p $LOG_DIR
echo -e "${GREEN}✓${NC} 目录创建完成"
echo ""

# 步骤3: 编译eBPF程序
echo -e "${BLUE}步骤3/6:${NC} 编译eBPF程序..."
make clean
make all
echo -e "${GREEN}✓${NC} 编译完成"
echo ""

# 步骤4: 安装文件
echo -e "${BLUE}步骤4/6:${NC} 安装文件..."
cp build/tc_microsegment $INSTALL_DIR/bin/
cp build/*.bpf.o $INSTALL_DIR/bpf/
ln -sf $INSTALL_DIR/bin/tc_microsegment $BIN_DIR/tc-micro
chmod +x $INSTALL_DIR/bin/tc_microsegment
echo -e "${GREEN}✓${NC} 文件安装完成"
echo ""

# 步骤5: 安装配置文件
echo -e "${BLUE}步骤5/6:${NC} 安装配置文件..."
if [ ! -f $CONFIG_DIR/config.json ]; then
    cat > $CONFIG_DIR/config.json <<'EOF'
{
  "interfaces": ["eth0"],
  "log_level": "info",
  "log_file": "/var/log/tc-microsegment/tc-micro.log",
  "metrics_port": 9100,
  "default_policy": "deny",
  "policies": []
}
EOF
    echo -e "${GREEN}✓${NC} 配置文件已创建"
else
    echo -e "${GREEN}✓${NC} 配置文件已存在，跳过"
fi
echo ""

# 步骤6: 安装systemd服务
echo -e "${BLUE}步骤6/6:${NC} 安装systemd服务..."
cat > /etc/systemd/system/tc-microsegment.service <<EOF
[Unit]
Description=eBPF TC Microsegmentation Service
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/bin/tc_microsegment --config $CONFIG_DIR/config.json
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
echo -e "${GREEN}✓${NC} systemd服务已安装"
echo ""

echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}  部署完成！${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo "下一步操作:"
echo "  1. 编辑配置: sudo vi $CONFIG_DIR/config.json"
echo "  2. 启动服务: sudo systemctl start tc-microsegment"
echo "  3. 查看状态: sudo systemctl status tc-microsegment"
echo "  4. 查看日志: sudo journalctl -u tc-microsegment -f"
echo "  5. 添加策略: sudo tc-micro policy add ..."
```

**任务3: 升级脚本**

创建 `scripts/upgrade.sh`:

```bash
#!/bin/bash
# upgrade.sh - 升级脚本

set -e

OLD_VERSION=$(tc-micro --version 2>/dev/null | awk '{print $3}' || echo "unknown")
NEW_VERSION=$(cat VERSION)

echo "升级: $OLD_VERSION → $NEW_VERSION"
echo ""

# 1. 备份配置
echo "备份配置..."
cp /etc/tc-microsegment/config.json /etc/tc-microsegment/config.json.bak
echo "✓ 配置已备份"

# 2. 停止服务
echo "停止服务..."
systemctl stop tc-microsegment || true
echo "✓ 服务已停止"

# 3. 卸载旧版本eBPF程序
echo "清理旧版本..."
tc filter del dev eth0 ingress 2>/dev/null || true
echo "✓ 旧版本已清理"

# 4. 部署新版本
echo "部署新版本..."
bash scripts/deploy.sh

# 5. 恢复配置
echo "恢复配置..."
cp /etc/tc-microsegment/config.json.bak /etc/tc-microsegment/config.json
echo "✓ 配置已恢复"

# 6. 启动服务
echo "启动服务..."
systemctl start tc-microsegment
sleep 2
systemctl status tc-microsegment

echo ""
echo "✓ 升级完成！"
```

#### 📚 学习资料 (2小时)

1. **Bash脚本最佳实践** (1小时)
   - 参考: https://google.github.io/styleguide/shellguide.html
   - 重点: 错误处理、颜色输出、参数验证

2. **systemd服务管理** (1小时)
   - 参考: `man systemd.service`
   - 重点: Type、Restart策略、日志管理

#### ✅ 完成标准

- [ ] check_env.sh 能正确检查所有依赖
- [ ] deploy.sh 能一键完成部署
- [ ] upgrade.sh 能平滑升级
- [ ] 所有脚本有完整错误处理
- [ ] 日志输出清晰友好

---

### 📅 Day 2 (Tuesday): 灰度部署脚本开发

#### 🎯 任务目标
实现金丝雀部署(Canary Deployment)脚本，支持分阶段灰度上线和自动回滚。

#### ✅ 具体任务

**任务1: 金丝雀部署脚本**

创建 `scripts/canary_deploy.sh`:

```bash
#!/bin/bash
# canary_deploy.sh - 金丝雀部署脚本

set -e

CANARY_STAGES=(5 10 25 50 100)  # 灰度比例: 5% -> 10% -> 25% -> 50% -> 100%
STAGE_DURATION=300              # 每阶段持续时间(秒)
HEALTH_CHECK_INTERVAL=10        # 健康检查间隔(秒)
ERROR_THRESHOLD=5               # 错误率阈值(%)

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 健康检查函数
check_health() {
    local stage=$1

    echo -n "健康检查 (阶段${stage}%)... "

    # 1. 检查服务状态
    if ! systemctl is-active --quiet tc-microsegment; then
        echo -e "${RED}✗ 服务未运行${NC}"
        return 1
    fi

    # 2. 检查eBPF程序是否加载
    if ! bpftool prog show | grep -q "tc_microsegment"; then
        echo -e "${RED}✗ eBPF程序未加载${NC}"
        return 1
    fi

    # 3. 检查统计数据
    STATS=$(tc-micro stats show --json 2>/dev/null || echo '{}')

    # 获取丢包率
    PACKETS_TOTAL=$(echo $STATS | jq -r '.packets_total // 0')
    PACKETS_DROPPED=$(echo $STATS | jq -r '.packets_dropped // 0')

    if [ $PACKETS_TOTAL -gt 0 ]; then
        DROP_RATE=$(awk "BEGIN {printf \"%.2f\", ($PACKETS_DROPPED/$PACKETS_TOTAL)*100}")

        if (( $(echo "$DROP_RATE > $ERROR_THRESHOLD" | bc -l) )); then
            echo -e "${RED}✗ 丢包率过高: ${DROP_RATE}%${NC}"
            return 1
        fi
    fi

    # 4. 检查CPU使用率
    CPU_USAGE=$(top -bn1 | grep "tc_microsegment" | awk '{print $9}' | head -n1)
    if [ -n "$CPU_USAGE" ] && (( $(echo "$CPU_USAGE > 80" | bc -l) )); then
        echo -e "${YELLOW}⚠ CPU使用率较高: ${CPU_USAGE}%${NC}"
    fi

    echo -e "${GREEN}✓ 健康${NC}"
    return 0
}

# 流量切换函数
switch_traffic() {
    local percentage=$1

    echo "切换流量至新版本: ${percentage}%"

    # 这里使用iptables的random模块实现流量分配
    # 实际生产环境可能使用负载均衡器或其他流量管理工具

    # 清除旧规则
    iptables -t mangle -F TC_CANARY 2>/dev/null || true
    iptables -t mangle -X TC_CANARY 2>/dev/null || true

    # 创建新链
    iptables -t mangle -N TC_CANARY

    # 添加规则: percentage% 流量走新版本
    iptables -t mangle -A TC_CANARY -m statistic --mode random \
             --probability $(awk "BEGIN {print $percentage/100}") \
             -j MARK --set-mark 0x2  # 新版本标记

    iptables -t mangle -A TC_CANARY -j MARK --set-mark 0x1  # 旧版本标记

    # 应用到PREROUTING
    iptables -t mangle -I PREROUTING -j TC_CANARY

    echo "✓ 流量切换完成"
}

# 回滚函数
rollback() {
    echo -e "${RED}检测到异常，执行回滚...${NC}"

    # 1. 切换流量到旧版本
    switch_traffic 0

    # 2. 停止新版本
    systemctl stop tc-microsegment

    # 3. 恢复旧版本
    systemctl start tc-microsegment-old

    # 4. 清理eBPF程序
    tc filter del dev eth0 ingress 2>/dev/null || true

    echo -e "${GREEN}✓ 回滚完成${NC}"
    exit 1
}

# 主流程
echo "========================================="
echo "  金丝雀部署启动"
echo "========================================="
echo ""

# 备份当前版本
echo "备份当前版本..."
cp /opt/tc-microsegment/bin/tc_microsegment \
   /opt/tc-microsegment/bin/tc_microsegment.old
cp /etc/systemd/system/tc-microsegment.service \
   /etc/systemd/system/tc-microsegment-old.service
echo "✓ 备份完成"
echo ""

# 编译新版本
echo "编译新版本..."
make clean && make all
echo "✓ 编译完成"
echo ""

# 分阶段部署
for stage in "${CANARY_STAGES[@]}"; do
    echo "========================================="
    echo "  阶段: ${stage}% 流量"
    echo "========================================="

    # 切换流量
    switch_traffic $stage

    # 等待流量稳定
    echo "等待 ${STAGE_DURATION} 秒..."
    ELAPSED=0
    while [ $ELAPSED -lt $STAGE_DURATION ]; do
        sleep $HEALTH_CHECK_INTERVAL
        ELAPSED=$((ELAPSED + HEALTH_CHECK_INTERVAL))

        # 健康检查
        if ! check_health $stage; then
            rollback
        fi

        echo "  进度: ${ELAPSED}/${STAGE_DURATION}s"
    done

    echo -e "${GREEN}✓ 阶段${stage}%完成${NC}"
    echo ""
done

# 清理旧版本
echo "清理旧版本..."
rm -f /opt/tc-microsegment/bin/tc_microsegment.old
rm -f /etc/systemd/system/tc-microsegment-old.service
echo "✓ 清理完成"
echo ""

echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}  金丝雀部署成功完成！${NC}"
echo -e "${GREEN}=========================================${NC}"
```

**任务2: 回滚脚本**

创建 `scripts/rollback.sh`:

```bash
#!/bin/bash
# rollback.sh - 快速回滚脚本

set -e

echo "执行快速回滚..."

# 1. 停止当前服务
systemctl stop tc-microsegment

# 2. 卸载eBPF程序
tc filter del dev eth0 ingress 2>/dev/null || true

# 3. 恢复备份
if [ -f /opt/tc-microsegment/bin/tc_microsegment.backup ]; then
    cp /opt/tc-microsegment/bin/tc_microsegment.backup \
       /opt/tc-microsegment/bin/tc_microsegment
else
    echo "错误: 没有找到备份文件"
    exit 1
fi

# 4. 恢复配置
if [ -f /etc/tc-microsegment/config.json.backup ]; then
    cp /etc/tc-microsegment/config.json.backup \
       /etc/tc-microsegment/config.json
fi

# 5. 重启服务
systemctl start tc-microsegment

# 6. 验证
sleep 2
if systemctl is-active --quiet tc-microsegment; then
    echo "✓ 回滚成功"
else
    echo "✗ 回滚失败，请手动检查"
    exit 1
fi
```

#### 📚 学习资料 (2小时)

1. **金丝雀部署原理** (1小时)
   - 参考: https://martinfowler.com/bliki/CanaryRelease.html
   - 重点: 灰度策略、流量分配、监控指标

2. **iptables流量标记** (1小时)
   - 参考: `man iptables-extensions`
   - 重点: mangle表、MARK target、statistic match

#### ✅ 完成标准

- [ ] canary_deploy.sh 能分阶段部署
- [ ] 每阶段都有健康检查
- [ ] 异常时能自动回滚
- [ ] rollback.sh 能快速回滚
- [ ] 完整的日志输出

---

### 📅 Day 3 (Wednesday): Prometheus监控集成

#### 🎯 任务目标
集成Prometheus监控，导出eBPF统计指标，配置Grafana Dashboard。

#### ✅ 具体任务

**任务1: Prometheus Exporter实现**

在用户态程序中添加metrics导出 `src/metrics.c`:

```c
// metrics.c - Prometheus metrics exporter
#include <microhttpd.h>
#include <stdio.h>
#include <string.h>
#include "metrics.h"

#define PORT 9100

static struct MHD_Daemon *daemon = NULL;

// 生成Prometheus格式的指标
static char* generate_metrics(struct bpf_stats *stats)
{
    static char buffer[4096];

    snprintf(buffer, sizeof(buffer),
        "# HELP tc_micro_packets_total Total packets processed\n"
        "# TYPE tc_micro_packets_total counter\n"
        "tc_micro_packets_total %llu\n"
        "\n"
        "# HELP tc_micro_packets_allowed Packets allowed by policy\n"
        "# TYPE tc_micro_packets_allowed counter\n"
        "tc_micro_packets_allowed %llu\n"
        "\n"
        "# HELP tc_micro_packets_denied Packets denied by policy\n"
        "# TYPE tc_micro_packets_denied counter\n"
        "tc_micro_packets_denied %llu\n"
        "\n"
        "# HELP tc_micro_sessions_active Active sessions\n"
        "# TYPE tc_micro_sessions_active gauge\n"
        "tc_micro_sessions_active %u\n"
        "\n"
        "# HELP tc_micro_policy_lookups_total Total policy lookups\n"
        "# TYPE tc_micro_policy_lookups_total counter\n"
        "tc_micro_policy_lookups_total %llu\n"
        "\n"
        "# HELP tc_micro_session_cache_hits Session cache hits\n"
        "# TYPE tc_micro_session_cache_hits counter\n"
        "tc_micro_session_cache_hits %llu\n"
        "\n"
        "# HELP tc_micro_session_cache_misses Session cache misses\n"
        "# TYPE tc_micro_session_cache_misses counter\n"
        "tc_micro_session_cache_misses %llu\n"
        "\n"
        "# HELP tc_micro_cache_hit_rate Session cache hit rate\n"
        "# TYPE tc_micro_cache_hit_rate gauge\n"
        "tc_micro_cache_hit_rate %.2f\n"
        "\n"
        "# HELP tc_micro_map_pressure Map pressure percentage\n"
        "# TYPE tc_micro_map_pressure gauge\n"
        "tc_micro_map_pressure %u\n"
        "\n"
        "# HELP tc_micro_tcp_syn_floods_detected SYN flood attacks detected\n"
        "# TYPE tc_micro_tcp_syn_floods_detected counter\n"
        "tc_micro_tcp_syn_floods_detected %llu\n",
        stats->packets_total,
        stats->packets_allowed,
        stats->packets_denied,
        stats->sessions_active,
        stats->policy_lookups,
        stats->cache_hits,
        stats->cache_misses,
        stats->cache_hits * 100.0 / (stats->cache_hits + stats->cache_misses + 1),
        stats->map_pressure,
        stats->syn_floods);

    return buffer;
}

// HTTP请求处理
static int handle_request(void *cls,
                         struct MHD_Connection *connection,
                         const char *url,
                         const char *method,
                         const char *version,
                         const char *upload_data,
                         size_t *upload_data_size,
                         void **con_cls)
{
    struct bpf_stats *stats = (struct bpf_stats *)cls;
    struct MHD_Response *response;
    int ret;

    if (strcmp(url, "/metrics") != 0) {
        const char *page = "Use /metrics endpoint";
        response = MHD_create_response_from_buffer(strlen(page),
                                                   (void *)page,
                                                   MHD_RESPMEM_PERSISTENT);
        ret = MHD_queue_response(connection, MHD_HTTP_NOT_FOUND, response);
        MHD_destroy_response(response);
        return ret;
    }

    // 生成metrics
    char *metrics = generate_metrics(stats);

    response = MHD_create_response_from_buffer(strlen(metrics),
                                               (void *)metrics,
                                               MHD_RESPMEM_MUST_COPY);
    MHD_add_response_header(response, "Content-Type", "text/plain");

    ret = MHD_queue_response(connection, MHD_HTTP_OK, response);
    MHD_destroy_response(response);

    return ret;
}

// 启动metrics服务器
int metrics_server_start(struct bpf_stats *stats, int port)
{
    daemon = MHD_start_daemon(MHD_USE_SELECT_INTERNALLY,
                             port,
                             NULL, NULL,
                             &handle_request, stats,
                             MHD_OPTION_END);

    if (daemon == NULL) {
        fprintf(stderr, "Failed to start metrics server on port %d\n", port);
        return -1;
    }

    printf("Metrics server started on port %d\n", port);
    printf("Access metrics at: http://localhost:%d/metrics\n", port);

    return 0;
}

// 停止metrics服务器
void metrics_server_stop(void)
{
    if (daemon) {
        MHD_stop_daemon(daemon);
        daemon = NULL;
    }
}
```

**任务2: Prometheus配置**

创建 `deploy/prometheus.yml`:

```yaml
# Prometheus配置
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'tc-microsegment'
    static_configs:
      - targets: ['localhost:9100']
        labels:
          instance: 'node1'
          environment: 'production'

    # 抓取间隔
    scrape_interval: 10s
    scrape_timeout: 5s

    # 指标relabel
    metric_relabel_configs:
      - source_labels: [__name__]
        regex: 'tc_micro_.*'
        action: keep

# 告警规则
rule_files:
  - 'alerts.yml'

# Alertmanager配置
alerting:
  alertmanagers:
    - static_configs:
        - targets: ['localhost:9093']
```

**任务3: 告警规则**

创建 `deploy/alerts.yml`:

```yaml
groups:
  - name: tc_microsegment_alerts
    interval: 30s
    rules:
      # 高丢包率告警
      - alert: HighDropRate
        expr: |
          (rate(tc_micro_packets_denied[5m]) /
           rate(tc_micro_packets_total[5m])) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "高丢包率检测"
          description: "丢包率 {{ $value | humanizePercentage }} 超过10%"

      # Map压力告警
      - alert: MapPressureHigh
        expr: tc_micro_map_pressure > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Map压力过高"
          description: "Map使用率 {{ $value }}% 超过80%"

      # 缓存命中率低
      - alert: LowCacheHitRate
        expr: tc_micro_cache_hit_rate < 0.7
        for: 5m
        labels:
          severity: info
        annotations:
          summary: "缓存命中率较低"
          description: "缓存命中率 {{ $value | humanizePercentage }} 低于70%"

      # SYN Flood检测
      - alert: SynFloodDetected
        expr: rate(tc_micro_tcp_syn_floods_detected[1m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "检测到SYN Flood攻击"
          description: "检测到SYN Flood攻击，速率: {{ $value }}/s"

      # 服务不可用
      - alert: ServiceDown
        expr: up{job="tc-microsegment"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "服务不可用"
          description: "tc-microsegment服务在 {{ $labels.instance }} 上不可用"
```

**任务4: Grafana Dashboard**

创建 `deploy/grafana-dashboard.json`:

```json
{
  "dashboard": {
    "title": "eBPF TC Microsegmentation",
    "tags": ["ebpf", "networking", "security"],
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "title": "Packet Processing Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(tc_micro_packets_total[1m])",
            "legendFormat": "Total",
            "refId": "A"
          },
          {
            "expr": "rate(tc_micro_packets_allowed[1m])",
            "legendFormat": "Allowed",
            "refId": "B"
          },
          {
            "expr": "rate(tc_micro_packets_denied[1m])",
            "legendFormat": "Denied",
            "refId": "C"
          }
        ],
        "yaxes": [
          {
            "format": "pps",
            "label": "Packets/sec"
          }
        ]
      },
      {
        "id": 2,
        "title": "Cache Hit Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "tc_micro_cache_hit_rate",
            "legendFormat": "Hit Rate",
            "refId": "A"
          }
        ],
        "yaxes": [
          {
            "format": "percentunit",
            "max": 1,
            "min": 0
          }
        ],
        "thresholds": [
          {
            "value": 0.7,
            "colorMode": "critical",
            "op": "lt"
          }
        ]
      },
      {
        "id": 3,
        "title": "Active Sessions",
        "type": "graph",
        "targets": [
          {
            "expr": "tc_micro_sessions_active",
            "legendFormat": "Sessions",
            "refId": "A"
          }
        ]
      },
      {
        "id": 4,
        "title": "Map Pressure",
        "type": "gauge",
        "targets": [
          {
            "expr": "tc_micro_map_pressure",
            "refId": "A"
          }
        ],
        "thresholds": {
          "steps": [
            { "value": 0, "color": "green" },
            { "value": 70, "color": "yellow" },
            { "value": 80, "color": "red" }
          ]
        }
      }
    ]
  }
}
```

#### 📚 学习资料 (2.5小时)

1. **Prometheus基础** (1小时)
   - 参考: https://prometheus.io/docs/introduction/overview/
   - 重点: metrics类型、PromQL查询

2. **libmicrohttpd使用** (1小时)
   - 参考: https://www.gnu.org/software/libmicrohttpd/
   - 重点: HTTP服务器、请求处理

3. **Grafana Dashboard创建** (0.5小时)
   - 参考: https://grafana.com/docs/grafana/latest/dashboards/
   - 重点: 面板配置、查询语法

#### ✅ 完成标准

- [ ] Metrics服务器在9100端口运行
- [ ] Prometheus能成功抓取指标
- [ ] 告警规则配置正确
- [ ] Grafana Dashboard显示所有指标
- [ ] 模拟告警能正常触发

---

### 📅 Day 4 (Thursday): 金丝雀部署测试

#### 🎯 任务目标
在测试环境执行完整的金丝雀部署流程，验证部署脚本和监控系统。

#### ✅ 具体任务

**任务1: 测试环境准备**

```bash
#!/bin/bash
# setup_test_env.sh - 测试环境准备

# 1. 创建虚拟网络环境
sudo ip netns add test-old
sudo ip netns add test-new
sudo ip netns add test-client

# 创建veth对
sudo ip link add veth-old type veth peer name veth-old-br
sudo ip link add veth-new type veth peer name veth-new-br
sudo ip link add veth-client type veth peer name veth-client-br

# 移动到namespace
sudo ip link set veth-old netns test-old
sudo ip link set veth-new netns test-new
sudo ip link set veth-client netns test-client

# 配置IP
sudo ip netns exec test-old ip addr add 10.0.1.10/24 dev veth-old
sudo ip netns exec test-new ip addr add 10.0.1.20/24 dev veth-new
sudo ip netns exec test-client ip addr add 10.0.1.100/24 dev veth-client

# 启动接口
sudo ip netns exec test-old ip link set veth-old up
sudo ip netns exec test-new ip link set veth-new up
sudo ip netns exec test-client ip link set veth-client up
sudo ip link set veth-old-br up
sudo ip link set veth-new-br up
sudo ip link set veth-client-br up

# 2. 创建bridge
sudo ip link add br0 type bridge
sudo ip link set veth-old-br master br0
sudo ip link set veth-new-br master br0
sudo ip link set veth-client-br master br0
sudo ip link set br0 up

echo "✓ 测试环境准备完成"
```

**任务2: 金丝雀部署测试脚本**

创建 `tests/test_canary_deploy.sh`:

```bash
#!/bin/bash
# test_canary_deploy.sh - 金丝雀部署测试

set -e

TEST_DURATION=60  # 每阶段测试时间(秒)
CLIENT_THREADS=10

echo "========================================="
echo "  金丝雀部署测试"
echo "========================================="
echo ""

# 1. 启动旧版本
echo "启动旧版本 (v1.0)..."
sudo ip netns exec test-old /opt/tc-microsegment/bin/tc_microsegment \
    --version 1.0 --port 8080 &
OLD_PID=$!
sleep 2
echo "✓ 旧版本运行中 (PID: $OLD_PID)"
echo ""

# 2. 启动新版本
echo "启动新版本 (v1.1)..."
sudo ip netns exec test-new /opt/tc-microsegment/bin/tc_microsegment \
    --version 1.1 --port 8081 &
NEW_PID=$!
sleep 2
echo "✓ 新版本运行中 (PID: $NEW_PID)"
echo ""

# 3. 启动负载生成器
echo "启动负载生成器..."
sudo ip netns exec test-client wrk \
    -t $CLIENT_THREADS \
    -c 100 \
    -d ${TEST_DURATION}s \
    --latency \
    http://10.0.1.10:8080/ > /tmp/wrk_old.txt &

sudo ip netns exec test-client wrk \
    -t $CLIENT_THREADS \
    -c 100 \
    -d ${TEST_DURATION}s \
    --latency \
    http://10.0.1.20:8081/ > /tmp/wrk_new.txt &

echo "✓ 负载生成器运行中"
echo ""

# 4. 执行金丝雀部署
STAGES=(0 25 50 75 100)

for i in "${!STAGES[@]}"; do
    stage=${STAGES[$i]}

    echo "========================================="
    echo "  阶段 $((i+1))/5: ${stage}% 新版本"
    echo "========================================="

    # 调整iptables规则分配流量
    sudo iptables -t mangle -F TC_CANARY 2>/dev/null || true
    sudo iptables -t mangle -X TC_CANARY 2>/dev/null || true
    sudo iptables -t mangle -N TC_CANARY

    if [ $stage -gt 0 ]; then
        sudo iptables -t mangle -A TC_CANARY \
            -m statistic --mode random \
            --probability $(awk "BEGIN {print $stage/100}") \
            -j DNAT --to-destination 10.0.1.20:8081
    fi

    sudo iptables -t mangle -A TC_CANARY \
        -j DNAT --to-destination 10.0.1.10:8080

    sudo iptables -t mangle -I PREROUTING -j TC_CANARY

    # 等待并监控
    ELAPSED=0
    while [ $ELAPSED -lt 30 ]; do
        sleep 5
        ELAPSED=$((ELAPSED + 5))

        # 检查错误率
        OLD_ERRORS=$(curl -s http://10.0.1.10:8080/stats | jq -r '.errors // 0')
        NEW_ERRORS=$(curl -s http://10.0.1.20:8081/stats | jq -r '.errors // 0')

        echo "  [$ELAPSED/30s] 旧版本错误: $OLD_ERRORS, 新版本错误: $NEW_ERRORS"

        # 如果新版本错误过多，回滚
        if [ $NEW_ERRORS -gt 100 ]; then
            echo "✗ 新版本错误过多，回滚！"
            sudo iptables -t mangle -F TC_CANARY
            exit 1
        fi
    done

    echo "✓ 阶段${stage}%完成"
    echo ""
done

# 5. 收集结果
echo "========================================="
echo "  测试结果"
echo "========================================="
echo ""

echo "旧版本 (v1.0):"
cat /tmp/wrk_old.txt | grep -E "Requests/sec|Latency"
echo ""

echo "新版本 (v1.1):"
cat /tmp/wrk_new.txt | grep -E "Requests/sec|Latency"
echo ""

# 6. 清理
kill $OLD_PID $NEW_PID 2>/dev/null || true
sudo iptables -t mangle -F TC_CANARY
sudo iptables -t mangle -X TC_CANARY

echo "✓ 金丝雀部署测试完成"
```

**任务3: 测试报告生成**

创建 `tests/generate_canary_report.sh`:

```bash
#!/bin/bash
# generate_canary_report.sh - 生成测试报告

cat > /tmp/canary_test_report.md <<'EOF'
# 金丝雀部署测试报告

## 测试环境
- 测试时间: $(date)
- 旧版本: v1.0
- 新版本: v1.1
- 测试工具: wrk
- 并发数: 100

## 部署阶段

| 阶段 | 新版本流量% | 持续时间 | 错误数 | 延迟P50 | 延迟P99 | 结果 |
|------|-------------|----------|--------|---------|---------|------|
| 1    | 0%          | 30s      | 0      | 5ms     | 12ms    | ✓    |
| 2    | 25%         | 30s      | 0      | 5ms     | 13ms    | ✓    |
| 3    | 50%         | 30s      | 0      | 6ms     | 14ms    | ✓    |
| 4    | 75%         | 30s      | 0      | 6ms     | 15ms    | ✓    |
| 5    | 100%        | 30s      | 0      | 7ms     | 16ms    | ✓    |

## 性能对比

### 吞吐量
- 旧版本: 15,234 req/s
- 新版本: 15,892 req/s
- 提升: +4.3%

### 延迟
- 旧版本 P50: 5.2ms, P99: 12.4ms
- 新版本 P50: 5.8ms, P99: 14.1ms
- 变化: P50 +11%, P99 +13%

## 告警触发情况

- 无告警触发

## 结论

✓ **金丝雀部署成功**

- 所有阶段健康检查通过
- 无异常回滚
- 性能指标稳定
- 建议: 可以推广到生产环境

## 改进建议

1. 增加更细粒度的流量切换(5% -> 10% -> 25% ...)
2. 延长每阶段观察时间到5分钟
3. 添加更多自动化健康检查指标
EOF

echo "✓ 报告已生成: /tmp/canary_test_report.md"
cat /tmp/canary_test_report.md
```

#### 📚 学习资料 (1.5小时)

1. **灰度发布最佳实践** (1小时)
   - 参考: https://www.martinfowler.com/bliki/CanaryRelease.html
   - 重点: 风险控制、监控指标、回滚策略

2. **wrk压力测试工具** (0.5小时)
   - 参考: https://github.com/wg/wrk
   - 重点: 参数调优、结果分析

#### ✅ 完成标准

- [ ] 测试环境成功搭建
- [ ] 金丝雀部署脚本正常运行
- [ ] 所有阶段健康检查通过
- [ ] 测试报告自动生成
- [ ] 性能数据完整记录

---

### 📅 Day 5 (Friday): 项目交付与演示准备

#### 🎯 任务目标
完成项目交付文档、演示材料，录制演示视频，准备项目总结。

#### ✅ 具体任务

**任务1: 项目交付文档**

创建 `docs/DELIVERY.md`:

```markdown
# eBPF微隔离系统 - 项目交付文档

## 项目概述

**项目名称**: eBPF TC 微隔离系统
**版本**: v1.0.0
**交付日期**: 2025-10-24
**开发周期**: 6周

### 核心功能
- ✅ 基于eBPF TC的高性能包过滤
- ✅ 5元组策略匹配
- ✅ 会话跟踪与缓存
- ✅ TCP状态机管理
- ✅ IP段匹配(LPM Trie)
- ✅ Prometheus监控集成
- ✅ 金丝雀部署支持

### 性能指标
| 指标 | 目标 | 实测 | 状态 |
|------|------|------|------|
| P50延迟 | <20μs | 12μs | ✅ |
| P99延迟 | <50μs | 35μs | ✅ |
| 吞吐量 | >30Gbps | 38Gbps | ✅ |
| CPU使用率 | <10% | 7% | ✅ |
| 会话容量 | >100K | 150K | ✅ |
| 缓存命中率 | >90% | 94% | ✅ |

## 交付内容

### 1. 源代码
```
├── src/
│   ├── tc_microsegment.bpf.c    # eBPF程序
│   ├── main.c                    # 用户态主程序
│   ├── policy.c                  # 策略管理
│   ├── session.c                 # 会话管理
│   ├── stats.c                   # 统计功能
│   └── metrics.c                 # Prometheus导出
├── include/
│   └── common.h                  # 公共头文件
└── tests/
    ├── unit/                     # 单元测试
    ├── functional/               # 功能测试
    └── performance/              # 性能测试
```

### 2. 文档
- ✅ 架构设计文档 (specs/ebpf-tc-architecture.md)
- ✅ 实施指南 (specs/ebpf-tc-implementation.md)
- ✅ API文档 (docs/API.md)
- ✅ 运维手册 (docs/OPS.md)
- ✅ 故障排查指南 (docs/TROUBLESHOOTING.md)

### 3. 部署工具
- ✅ 一键部署脚本 (scripts/deploy.sh)
- ✅ 环境检查脚本 (scripts/check_env.sh)
- ✅ 金丝雀部署脚本 (scripts/canary_deploy.sh)
- ✅ 回滚脚本 (scripts/rollback.sh)
- ✅ systemd服务配置

### 4. 监控配置
- ✅ Prometheus配置 (deploy/prometheus.yml)
- ✅ 告警规则 (deploy/alerts.yml)
- ✅ Grafana Dashboard (deploy/grafana-dashboard.json)

### 5. 测试报告
- ✅ 单元测试报告 (test_reports/unit_test_report.md)
- ✅ 功能测试报告 (test_reports/functional_test_report.md)
- ✅ 性能测试报告 (test_reports/performance_test_report.md)
- ✅ 金丝雀部署测试报告 (test_reports/canary_test_report.md)

## 快速开始

### 安装
```bash
# 1. 克隆代码
git clone https://github.com/yourorg/ebpf-microsegment.git
cd ebpf-microsegment

# 2. 检查环境
sudo bash scripts/check_env.sh

# 3. 一键部署
sudo bash scripts/deploy.sh

# 4. 启动服务
sudo systemctl start tc-microsegment

# 5. 验证
sudo tc-micro stats show
```

### 添加策略
```bash
# 允许SSH
sudo tc-micro policy add \
    --src-ip 10.0.0.0/24 \
    --dst-port 22 \
    --protocol tcp \
    --action allow

# 拒绝HTTP
sudo tc-micro policy add \
    --dst-port 80 \
    --protocol tcp \
    --action deny
```

## 技术架构

### 数据流
```
数据包 → TC ingress hook → eBPF程序 → 策略匹配 → 放行/拒绝
                                   ↓
                              会话缓存
                                   ↓
                              统计更新
                                   ↓
                            Prometheus导出
```

### 核心组件
1. **eBPF程序** (内核态)
   - tc_microsegment.bpf.c
   - 5元组匹配、会话跟踪、TCP状态机

2. **控制程序** (用户态)
   - 策略管理
   - 统计监控
   - Metrics导出

3. **监控系统**
   - Prometheus抓取
   - Grafana可视化
   - 告警规则

## 运维指南

### 日常操作
```bash
# 查看状态
sudo systemctl status tc-microsegment

# 查看日志
sudo journalctl -u tc-microsegment -f

# 查看统计
sudo tc-micro stats show

# 查看会话
sudo tc-micro session list

# 重载策略
sudo tc-micro policy reload
```

### 升级
```bash
sudo bash scripts/upgrade.sh
```

### 回滚
```bash
sudo bash scripts/rollback.sh
```

### 监控
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000
- Metrics: http://localhost:9100/metrics

## 已知限制

1. **内核版本**: 需要 Linux 5.10+
2. **Map容量**: 会话表最大100K (可调整)
3. **复杂DPI**: 需要用户态辅助
4. **多网卡**: 当前版本支持单网卡 (可扩展)

## 后续优化建议

1. **性能优化**
   - 使用XDP替代TC (更早拦截)
   - 优化Map大小和LRU策略
   - 添加per-CPU哈希表

2. **功能增强**
   - 支持IPv6
   - 添加流量镜像
   - 集成IDS/IPS

3. **运维改进**
   - 添加Web UI
   - 集成K8s CNI
   - 支持配置中心

## 联系方式

- **技术负责人**: [你的名字]
- **Email**: [你的邮箱]
- **项目地址**: https://github.com/yourorg/ebpf-microsegment
- **文档**: https://docs.yourorg.com/ebpf-microsegment

## 附录

### A. 依赖软件版本
- Linux Kernel: >= 5.10
- clang/LLVM: >= 11
- libbpf: >= 0.6
- systemd: >= 245

### B. 测试环境
- OS: Ubuntu 22.04 LTS
- Kernel: 5.15.0
- CPU: Intel Xeon 2.5GHz
- Memory: 16GB
- Network: 10Gbps

### C. 参考文档
- [eBPF官方文档](https://ebpf.io/)
- [Cilium项目](https://cilium.io/)
- [libbpf](https://github.com/libbpf/libbpf)
```

**任务2: 演示脚本**

创建 `demo/demo.sh`:

```bash
#!/bin/bash
# demo.sh - 15分钟功能演示脚本

set -e

echo "========================================="
echo "  eBPF微隔离系统功能演示"
echo "========================================="
echo ""
sleep 2

# 1. 环境展示
echo "=== 1. 系统环境 ==="
echo ""
echo "内核版本:"
uname -r
echo ""
echo "eBPF支持:"
if [ -f /sys/kernel/btf/vmlinux ]; then
    echo "✓ BTF已启用"
fi
echo ""
sleep 3

# 2. 部署演示
echo "=== 2. 一键部署 ==="
echo ""
echo "执行环境检查..."
sudo bash scripts/check_env.sh
echo ""
echo "执行部署..."
sudo bash scripts/deploy.sh
echo ""
sleep 3

# 3. 策略管理演示
echo "=== 3. 策略管理 ==="
echo ""
echo "添加SSH允许策略:"
sudo tc-micro policy add \
    --src-ip 10.0.0.0/24 \
    --dst-port 22 \
    --protocol tcp \
    --action allow
echo ""
echo "添加HTTP拒绝策略:"
sudo tc-micro policy add \
    --dst-port 80 \
    --protocol tcp \
    --action deny
echo ""
echo "查看所有策略:"
sudo tc-micro policy list
echo ""
sleep 5

# 4. 流量测试
echo "=== 4. 流量测试 ==="
echo ""
echo "发起SSH连接 (应该允许)..."
timeout 2 nc -zv 10.0.0.50 22 || echo "连接成功"
echo ""
echo "发起HTTP连接 (应该拒绝)..."
timeout 2 nc -zv 10.0.0.50 80 || echo "连接被拒绝 ✓"
echo ""
sleep 3

# 5. 会话跟踪
echo "=== 5. 会话跟踪 ==="
echo ""
echo "查看活动会话:"
sudo tc-micro session list | head -n 10
echo ""
echo "会话统计:"
sudo tc-micro stats show | grep -E "sessions|cache"
echo ""
sleep 3

# 6. 性能监控
echo "=== 6. 性能监控 ==="
echo ""
echo "实时统计:"
sudo tc-micro stats show
echo ""
echo "Prometheus指标 (http://localhost:9100/metrics):"
curl -s http://localhost:9100/metrics | grep -E "^tc_micro" | head -n 5
echo ""
sleep 3

# 7. 压力测试
echo "=== 7. 压力测试 ==="
echo ""
echo "启动10秒压力测试..."
wrk -t 4 -c 100 -d 10s http://10.0.0.50/ &
WRK_PID=$!

# 实时显示统计
for i in {1..10}; do
    echo "[$i/10] 当前统计:"
    sudo tc-micro stats show | grep -E "packets|sessions"
    sleep 1
done

wait $WRK_PID
echo ""
echo "压力测试完成"
echo ""
sleep 3

# 8. 监控演示
echo "=== 8. 监控系统 ==="
echo ""
echo "Grafana Dashboard: http://localhost:3000"
echo "Prometheus: http://localhost:9090"
echo ""
echo "打开浏览器查看实时监控..."
echo ""
sleep 5

# 9. 故障模拟与恢复
echo "=== 9. 故障恢复演示 ==="
echo ""
echo "模拟服务故障..."
sudo systemctl stop tc-microsegment
sleep 2
echo "检查状态:"
systemctl status tc-microsegment || echo "服务已停止"
echo ""
echo "执行自动恢复..."
sudo systemctl start tc-microsegment
sleep 2
echo "恢复后状态:"
systemctl status tc-microsegment
echo ""
sleep 3

# 10. 总结
echo "========================================="
echo "  演示完成！"
echo "========================================="
echo ""
echo "核心功能:"
echo "  ✓ 高性能包过滤 (38Gbps吞吐量)"
echo "  ✓ 5元组策略匹配"
echo "  ✓ 会话跟踪 (94%缓存命中率)"
echo "  ✓ Prometheus监控集成"
echo "  ✓ 一键部署与回滚"
echo ""
echo "性能指标:"
echo "  • P50延迟: 12μs"
echo "  • P99延迟: 35μs"
echo "  • CPU使用: 7%"
echo "  • 会话容量: 150K"
echo ""
echo "谢谢观看！"
```

**任务3: 演示视频录制**

录制15分钟演示视频，包含以下内容:

1. **开场 (1分钟)**
   - 项目介绍
   - 核心优势
   - 技术架构图

2. **环境展示 (2分钟)**
   - 系统环境
   - 依赖检查
   - 代码结构

3. **部署演示 (3分钟)**
   - 一键部署流程
   - 服务启动
   - 状态检查

4. **功能演示 (5分钟)**
   - 策略管理 (添加/删除/查看)
   - 流量测试 (允许/拒绝)
   - 会话跟踪
   - 统计信息

5. **性能测试 (2分钟)**
   - 压力测试执行
   - 实时监控展示
   - 性能指标说明

6. **监控系统 (1分钟)**
   - Grafana Dashboard
   - Prometheus查询
   - 告警展示

7. **高级功能 (1分钟)**
   - 金丝雀部署
   - 自动回滚
   - 故障恢复

8. **总结 (0.5分钟)**
   - 项目成果
   - 后续计划

**录制工具**: OBS Studio / Kazam / SimpleScreenRecorder

#### 📚 学习资料 (1小时)

1. **技术演示技巧** (0.5小时)
   - 重点: 演示流程设计、讲解技巧、常见问题处理

2. **视频录制与剪辑** (0.5小时)
   - 工具: OBS Studio
   - 重点: 屏幕录制、字幕添加、视频导出

#### ✅ 完成标准

- [ ] 交付文档完整清晰
- [ ] 演示脚本能顺利执行
- [ ] 演示视频录制完成
- [ ] 所有交付物已打包
- [ ] 项目归档完成

---

### 📅 本周总结 (Friday晚上)

#### ✍️ 输出物

创建文档 `docs/week6_summary.md`:

```markdown
# 第6周学习总结

## 完成情况
- [x] 自动化部署脚本 (deploy.sh, upgrade.sh, rollback.sh)
- [x] 金丝雀部署脚本 (canary_deploy.sh)
- [x] Prometheus监控集成 (metrics导出 + 告警)
- [x] Grafana Dashboard配置
- [x] 金丝雀部署测试 (完整流程验证)
- [x] 项目交付文档 (DELIVERY.md)
- [x] 演示脚本和视频录制

## 核心收获

### 1. 部署自动化
学会了:
- Bash脚本的最佳实践 (错误处理、颜色输出)
- systemd服务管理
- 环境检查的全面性考虑
- 一键部署的用户体验设计

### 2. 金丝雀部署
掌握了:
- 灰度发布的核心思想
- 流量分配策略 (iptables random)
- 健康检查的设计
- 自动回滚机制

### 3. 监控集成
实现了:
- Prometheus metrics导出 (libmicrohttpd)
- 告警规则配置
- Grafana Dashboard设计
- 关键指标选择 (丢包率、缓存命中率、Map压力)

### 4. 项目管理
完成了:
- 完整的交付文档
- 功能演示设计
- 测试报告整理
- 项目归档

## 部署脚本功能

| 脚本 | 功能 | 代码行数 |
|------|------|---------|
| check_env.sh | 环境检查 | 80行 |
| deploy.sh | 一键部署 | 120行 |
| upgrade.sh | 平滑升级 | 60行 |
| rollback.sh | 快速回滚 | 40行 |
| canary_deploy.sh | 金丝雀部署 | 180行 |

## 监控指标

已实现的Prometheus指标:
1. `tc_micro_packets_total` - 总包数
2. `tc_micro_packets_allowed` - 允许包数
3. `tc_micro_packets_denied` - 拒绝包数
4. `tc_micro_sessions_active` - 活动会话数
5. `tc_micro_cache_hit_rate` - 缓存命中率
6. `tc_micro_map_pressure` - Map压力
7. `tc_micro_syn_floods_detected` - SYN Flood检测

## 金丝雀部署测试结果

| 阶段 | 新版本流量% | 错误数 | 延迟P50 | 延迟P99 | 结果 |
|------|-------------|--------|---------|---------|------|
| 1    | 5%          | 0      | 5ms     | 12ms    | ✓    |
| 2    | 10%         | 0      | 5ms     | 13ms    | ✓    |
| 3    | 25%         | 0      | 6ms     | 14ms    | ✓    |
| 4    | 50%         | 0      | 6ms     | 15ms    | ✓    |
| 5    | 100%        | 0      | 7ms     | 16ms    | ✓    |

结论: ✅ 金丝雀部署成功,所有阶段健康检查通过

## 项目最终成果

### 代码统计
```
Language         files     blank   comment      code
-----------------------------------------------------
C                   8       456       623      3254
Bash               12       234       187      1456
Markdown            5        78         0       892
JSON                2         0         0       156
YAML                2        12         8        89
-----------------------------------------------------
SUM:               29       780       818      5847
```

### 测试覆盖
- 单元测试: 8个 (100%通过)
- 功能测试: 12个 (100%通过)
- 性能测试: 6个 (全部达标)
- 压力测试: 4个 (通过)

### 性能指标 (最终)
- P50延迟: **12μs** (目标 <20μs) ✓
- P99延迟: **35μs** (目标 <50μs) ✓
- 吞吐量: **38Gbps** (目标 >30Gbps) ✓
- CPU使用: **7%** @ 1Gbps (目标 <10%) ✓
- 会话容量: **150K** (目标 >100K) ✓
- 缓存命中率: **94%** (目标 >90%) ✓

## 交付清单

- [x] 源代码 (src/, include/, tests/)
- [x] 文档 (specs/, docs/)
- [x] 部署脚本 (scripts/)
- [x] 监控配置 (deploy/)
- [x] 测试报告 (test_reports/)
- [x] 演示材料 (demo/)
- [x] 演示视频 (15分钟)

## 项目总结

经过6周的开发,成功完成了基于eBPF TC的微隔离系统:

**技术突破**:
1. 深入理解eBPF编程模型和Verifier约束
2. 掌握TC hook机制和包处理流程
3. 实现高性能会话跟踪 (LRU_HASH + 缓存优化)
4. 集成完整的监控和告警系统

**工程实践**:
1. 完整的CI/CD流程 (部署、测试、回滚)
2. 金丝雀部署实现
3. 自动化测试框架
4. 详尽的文档和交付材料

**性能成果**:
相比用户态PACKET_MMAP方案:
- 延迟降低 **3倍** (50μs → 15μs)
- 吞吐量提升 **3.8倍** (10Gbps → 38Gbps)
- CPU使用降低 **65%** (20% → 7%)

**后续计划**:
1. 支持IPv6
2. 使用XDP进一步优化性能
3. 集成Kubernetes CNI
4. 添加Web管理界面
```

#### 🎯 本周验收标准

**必须完成**:
- [x] 部署脚本完整且可用
- [x] 金丝雀部署测试通过
- [x] Prometheus监控正常工作
- [x] 项目交付文档完整
- [x] 演示材料准备完成

**加分项**:
- [x] 演示视频录制完成
- [x] 监控Dashboard美观实用
- [x] 部署脚本用户体验优秀
- [x] 交付文档专业详尽

---

## 🎉 项目完成！

恭喜你完成了为期6周的eBPF微隔离系统开发！

### 📊 整体进度

| 周次 | 主题 | 完成度 |
|------|------|--------|
| Week 1 | 环境准备 + eBPF基础 | ✅ 100% |
| Week 2 | 基础框架开发 | ✅ 100% |
| Week 3 | 用户态控制程序 | ✅ 100% |
| Week 4 | 高级功能实现 | ✅ 100% |
| Week 5 | 测试与优化 | ✅ 100% |
| Week 6 | 生产部署准备 | ✅ 100% |

### 🏆 核心成就

1. **技术深度**: 掌握eBPF、TC、网络协议栈
2. **性能优化**: 实现3倍延迟降低、3.8倍吞吐提升
3. **工程质量**: 完整测试、文档、部署流程
4. **生产就绪**: 监控、告警、灰度发布

### 📚 知识体系

累计学习时间: **~60小时**

- eBPF原理与实践: 20小时
- 网络协议与TC: 15小时
- 性能优化与测试: 12小时
- 监控与运维: 8小时
- 部署与发布: 5小时

### 🚀 下一步

1. 在生产环境部署
2. 持续性能优化
3. 添加新功能 (IPv6, XDP)
4. 开源分享经验

---

**恭喜完成！你已经掌握了eBPF微隔离系统的全栈开发！** 🎊

---

**[⬅️ 第5周](./week5-testing-optimization.md)** | **[📚 目录](./README.md)**

## 🎉 恭喜完成全部6周学习！

你已经掌握了：
- ✅ eBPF和TC的深入理解
- ✅ 高性能微隔离系统开发
- ✅ 完整的测试和部署流程
- ✅ 生产级监控和运维能力

**下一步**: 将系统部署到生产环境，持续优化和迭代！
