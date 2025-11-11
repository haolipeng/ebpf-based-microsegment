# Spec Delta: 策略执行 - TC Egress Hook 支持

## MODIFIED Requirements

### Requirement: TC 集成

系统必须(SHALL)附加到 Linux 流量控制（TC）子系统的 **Ingress 和 Egress** 两个方向：
- 附加点：TC ingress hook 和 TC egress hook（TCX）
- 网络接口：可配置（例如，eth0、lo）
- 返回码：TC_ACT_OK（允许）或 TC_ACT_SHOT（丢弃）

**变更说明**: 从仅支持 TC ingress 扩展为支持双向（ingress + egress）。

#### Scenario: TC Ingress 程序附加

**Given** 用户空间代理已启动
**When** 调用 DataPlane.LoadAndAttach()
**Then** eBPF 程序必须(SHALL)附加到指定的网络接口的 **ingress** 方向
**And** 附加必须(SHALL)使用 TC ingress（TC_INGRESS 方向）
**And** 接口上的所有传入数据包必须(SHALL)被处理

#### Scenario: TC Egress 程序附加 ✅ 新增

**Given** 用户空间代理已启动
**When** 调用 DataPlane.LoadAndAttach()
**Then** eBPF 程序必须(SHALL)附加到指定的网络接口的 **egress** 方向
**And** 附加必须(SHALL)使用 TC egress（TC_EGRESS 方向）
**And** 接口上的所有传出数据包必须(SHALL)被处理
**And** 如果 egress attach 失败，必须(SHALL)清理已附加的 ingress hook

#### Scenario: 双向 Hook 卸载

**Given** TC ingress 和 egress hook 已附加
**When** 调用 DataPlane.Unload()
**Then** 系统必须(SHALL)卸载 egress hook
**And** 系统必须(SHALL)卸载 ingress hook
**And** 即使部分卸载失败，也必须(SHALL)尝试清理所有资源

## ADDED Requirements

### Requirement: 方向感知执行

系统必须(SHALL)根据数据包方向执行策略：
- **Ingress 方向**: 从外部进入容器/工作负载的流量
- **Egress 方向**: 从容器/工作负载发出的流量
- **方向检测**: 使用 `skb->ingress_ifindex` 字段判断（非零表示 ingress，零表示 egress）

#### Scenario: Ingress 流量方向检测

**Given** 数据包从外部网络到达容器接口
**When** TC ingress hook 处理数据包
**Then** 系统必须(SHALL)检测方向为 DIR_INGRESS
**And** `skb->ingress_ifindex` 必须(SHALL)非零
**And** 策略匹配必须(SHALL)使用 direction=INGRESS 或 direction=ANY

#### Scenario: Egress 流量方向检测

**Given** 数据包从容器发往外部网络
**When** TC egress hook 处理数据包
**Then** 系统必须(SHALL)检测方向为 DIR_EGRESS
**And** `skb->ingress_ifindex` 必须(SHALL)为零
**And** 策略匹配必须(SHALL)使用 direction=EGRESS 或 direction=ANY

#### Scenario: Egress 策略拒绝未授权出站连接

**Given** 存在 egress deny 策略：src=192.168.1.10、dst=8.8.8.8、dport=53、direction=EGRESS、action=DENY
**When** 容器 192.168.1.10 尝试向 8.8.8.8:53 发送 DNS 查询
**Then** 数据包必须(SHALL)在 egress hook 被拦截
**And** 数据包必须(SHALL)被丢弃（返回 TC_ACT_SHOT）
**And** STATS_EGRESS_DENIED 计数器必须(SHALL)递增
**And** 必须(SHALL)发送包含 direction=EGRESS 的流事件

#### Scenario: Ingress 允许但 Egress 拒绝的双向策略

**Given** 存在 ingress allow 策略：dst=192.168.1.10、dport=80、direction=INGRESS、action=ALLOW
**And** 存在 egress deny 策略：src=192.168.1.10、dst=0.0.0.0/0、direction=EGRESS、action=DENY
**When** 外部主机连接到 192.168.1.10:80
**Then** ingress 流量必须(SHALL)被允许
**When** 192.168.1.10 尝试主动连接外部服务
**Then** egress 流量必须(SHALL)被拒绝
**And** 两个方向的策略必须(SHALL)独立执行

### Requirement: 分方向统计

系统必须(SHALL)提供分方向的统计指标：
- **STAT_INGRESS_PACKETS**: Ingress 总数据包数
- **STAT_EGRESS_PACKETS**: Egress 总数据包数
- **STAT_INGRESS_DENIED**: Ingress 拒绝的数据包数
- **STAT_EGRESS_DENIED**: Egress 拒绝的数据包数

#### Scenario: Egress 统计更新

**Given** egress deny 策略已配置
**When** egress 数据包被拒绝
**Then** STAT_EGRESS_PACKETS 必须(SHALL)递增
**And** STAT_EGRESS_DENIED 必须(SHALL)递增
**And** STAT_DENIED_PACKETS（总计）必须(SHALL)递增

#### Scenario: Ingress 统计更新

**Given** ingress deny 策略已配置
**When** ingress 数据包被拒绝
**Then** STAT_INGRESS_PACKETS 必须(SHALL)递增
**And** STAT_INGRESS_DENIED 必须(SHALL)递增
**And** STAT_DENIED_PACKETS（总计）必须(SHALL)递增

### Requirement: TCX 和 Legacy TC 双向支持

系统必须(SHALL)同时支持 TCX 和 Legacy TC 两种模式的双向 attach：

#### Scenario: TCX 双向 attach

**Given** 内核版本 >= 6.6（支持 TCX）
**When** 系统初始化
**Then** 必须(SHALL)使用 `link.AttachTCX()` 附加 ingress hook
**And** 必须(SHALL)使用 `link.AttachTCX()` 附加 egress hook
**And** 两个 hook 必须(SHALL)使用同一个 eBPF 程序

#### Scenario: Legacy TC 双向 attach

**Given** 内核版本 < 6.6（不支持 TCX）
**When** 系统初始化
**Then** 必须(SHALL)使用 netlink API 添加 clsact qdisc
**And** 必须(SHALL)添加 ingress filter（parent=HANDLE_MIN_INGRESS）
**And** 必须(SHALL)添加 egress filter（parent=HANDLE_MIN_EGRESS）
**And** 两个 filter 必须(SHALL)使用同一个 eBPF 程序

## 向后兼容性

- 旧版本的 ingress-only 配置必须(SHALL)继续工作
- 如果未配置 direction，默认(SHALL)使用 direction=ANY（双向匹配）
- 现有统计指标必须(SHALL)保持含义不变

---

**变更 ID**: add-tc-egress-support
**修改日期**: 2025-11-11
**状态**: Proposed
