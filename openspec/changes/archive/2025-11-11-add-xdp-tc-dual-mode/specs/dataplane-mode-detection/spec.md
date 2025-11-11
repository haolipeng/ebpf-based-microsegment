# Spec: dataplane-mode-detection

## ADDED Requirements

### Requirement: 系统能力检测
系统 SHALL 能够检测内核版本、XDP 支持、TC 支持和网卡能力,以确定可用的数据平面模式。

#### Scenario: 检测支持所有模式的系统
- **GIVEN** 系统运行 kernel 6.6+,网卡驱动支持 Native XDP
- **WHEN** 调用 `DetectCapabilities("eth0")`
- **THEN** 返回的 Capabilities 显示:
  - `SupportsNativeXDP = true`
  - `SupportsGenericXDP = true`
  - `SupportsTCX = true`
  - `SupportsLegacyTC = true`
  - `KernelMajor = 6, KernelMinor >= 6`

#### Scenario: 检测只支持 Legacy TC 的旧系统
- **GIVEN** 系统运行 kernel 4.18
- **WHEN** 调用 `DetectCapabilities("eth0")`
- **THEN** 返回的 Capabilities 显示:
  - `SupportsNativeXDP = false`
  - `SupportsGenericXDP = false`
  - `SupportsTCX = false`
  - `SupportsLegacyTC = true`

#### Scenario: 检测失败时返回错误
- **GIVEN** 无法获取内核版本
- **WHEN** 调用 `DetectCapabilities("eth0")`
- **THEN** 返回错误 "failed to get kernel version"

---

### Requirement: 智能模式选择
系统 SHALL 根据检测到的能力和用户配置,自动选择最佳的数据平面模式,优先级为: Native XDP > Generic XDP > TCX > Legacy TC。

#### Scenario: 自动选择 Native XDP
- **GIVEN** Capabilities 显示支持 Native XDP
- **AND** ModeConfig.PreferXDP = true
- **AND** ModeConfig.ForceMode = ModeUnknown (auto)
- **WHEN** 调用 `SelectBestMode(caps, config)`
- **THEN** 返回 `ModeNativeXDP`

#### Scenario: 回退到 Generic XDP
- **GIVEN** Capabilities 显示不支持 Native XDP,但支持 Generic XDP
- **AND** ModeConfig.PreferXDP = true
- **AND** ModeConfig.AllowGenericXDP = true
- **WHEN** 调用 `SelectBestMode(caps, config)`
- **THEN** 返回 `ModeGenericXDP`

#### Scenario: 回退到 TCX
- **GIVEN** Capabilities 显示不支持任何 XDP,但支持 TCX
- **WHEN** 调用 `SelectBestMode(caps, config)`
- **THEN** 返回 `ModeTCX`

#### Scenario: 最终回退到 Legacy TC
- **GIVEN** Capabilities 显示只支持 Legacy TC
- **WHEN** 调用 `SelectBestMode(caps, config)`
- **THEN** 返回 `ModeLegacyTC`

#### Scenario: 没有可用模式
- **GIVEN** Capabilities 显示不支持任何模式
- **WHEN** 调用 `SelectBestMode(caps, config)`
- **THEN** 返回 `ModeUnknown`

---

### Requirement: 强制模式验证
系统 SHALL 支持用户强制指定数据平面模式,并验证该模式在当前系统上是否可用。

#### Scenario: 强制模式可用
- **GIVEN** ModeConfig.ForceMode = ModeNativeXDP
- **AND** Capabilities 显示支持 Native XDP
- **WHEN** 调用 `SelectBestMode(caps, config)`
- **THEN** 返回 `ModeNativeXDP`
- **AND** 日志输出 "Using forced mode: Native XDP"

#### Scenario: 强制模式不可用回退自动选择
- **GIVEN** ModeConfig.ForceMode = ModeNativeXDP
- **AND** Capabilities 显示不支持 Native XDP
- **WHEN** 调用 `SelectBestMode(caps, config)`
- **THEN** 返回自动选择的模式 (Generic XDP 或 TC)
- **AND** 日志输出警告 "Forced mode Native XDP not available, falling back to auto-select"

---

### Requirement: Native XDP 驱动测试
系统 SHALL 通过实际附加测试验证网卡驱动是否支持 Native XDP,而不仅仅依赖内核特性检测。

#### Scenario: 驱动支持 Native XDP
- **GIVEN** 网卡驱动实现了 XDP 支持
- **WHEN** 调用 `testNativeXDPSupport("eth0")`
- **THEN** 创建测试 XDP 程序
- **AND** 尝试以 DRV mode 附加到 eth0
- **AND** 附加成功,返回 true
- **AND** 清理测试程序

#### Scenario: 驱动不支持 Native XDP
- **GIVEN** 网卡驱动未实现 XDP 支持
- **WHEN** 调用 `testNativeXDPSupport("eth0")`
- **THEN** 尝试附加失败,返回 false
- **AND** 清理测试程序

---

### Requirement: 配置解析和验证
系统 SHALL 正确解析 DataPlane 配置选项,并设置合理的默认值。

#### Scenario: 使用默认配置
- **GIVEN** 配置文件未指定 dataplane 配置
- **WHEN** 加载配置
- **THEN** DataPlaneConfig 为:
  - `Mode = "auto"`
  - `PreferXDP = true`
  - `AllowGenericXDP = true`

#### Scenario: 解析用户配置
- **GIVEN** 配置文件指定:
  ```yaml
  dataplane:
    mode: xdp-native
    prefer_xdp: false
    allow_generic_xdp: false
  ```
- **WHEN** 加载配置
- **THEN** DataPlaneConfig 为:
  - `Mode = "xdp-native"`
  - `PreferXDP = false`
  - `AllowGenericXDP = false`

#### Scenario: 无效的模式字符串
- **GIVEN** 配置文件指定 `mode: invalid-mode`
- **WHEN** 加载配置
- **THEN** 返回配置验证错误
- **AND** 提示有效的模式选项

---

### Requirement: 日志和可观测性
系统 SHALL 为模式检测过程提供清晰的日志输出,帮助用户理解选择的模式和原因。

#### Scenario: 记录能力检测结果
- **WHEN** 执行能力检测
- **THEN** 日志输出包含:
  - 内核版本信息
  - 每项能力的检测结果 (✓/✗ 标记)
  - 最终选择的模式和原因

#### Scenario: 记录模式选择决策
- **WHEN** 选择 Native XDP 模式
- **THEN** 日志输出 "Selected mode: Native XDP (best performance)"

#### Scenario: 记录降级原因
- **WHEN** 无法使用 Native XDP,降级到 Generic XDP
- **THEN** 日志输出:
  - "✗ Native XDP not supported on eth0 (fallback to Generic)"
  - "Selected mode: Generic XDP (good performance)"
