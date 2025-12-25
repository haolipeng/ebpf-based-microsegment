 📋 方案概要

  核心能力

  1. 协议识别：支持 10+ 种应用层协议
    - Web：HTTP、HTTPS/TLS
    - 基础设施：DNS、SSH
    - 数据库：MySQL、PostgreSQL、Redis、MongoDB
    - 消息队列：Kafka、RabbitMQ
    - RPC：gRPC
  2. 细粒度策略：基于协议的访问控制
    - 示例：仅允许 HTTPS，拒绝 HTTP
    - 协议特性过滤（加密/明文、二进制/文本）
  3. 协议滥用检测：识别异常行为
    - SSH 隧道检测
    - DNS 隧道检测
    - 非标准端口上的协议
  4. 增强可视化：协议级别的流量分析

  技术架构

  两阶段检测策略：
  1. 阶段 1：端口启发式（快速，置信度 60%）
    - 80/8080 → HTTP
    - 443/8443 → HTTPS
    - 3306 → MySQL
  2. 阶段 2：Payload 特征匹配（精确，置信度 85-99%）
    - HTTP 方法识别（GET、POST、PUT）
    - TLS 握手检测（ClientHello/ServerHello）
    - DNS 头部验证
    - MySQL 协议版本检查

  性能优化

  - 延迟识别：仅在前几个数据包检测
  - 置信度缓存：高置信度后停止检测
  - 采样机制：可配置检测间隔（每 N 个包）
  - Payload 限制：默认仅检查前 128 字节
  - 预期开销：<5% 数据包处理开销

  实现路线图

  总工期：11-16 天

  Phase 1: 基础框架（3-4 天）
  - 数据结构扩展
  - 核心检测框架
  - 策略引擎集成

  Phase 2: 协议检测器（4-5 天）
  - HTTP/HTTPS 检测
  - DNS 检测
  - SSH 和数据库协议检测

  Phase 3: 用户态集成（3-4 天）
  - Go API 实现
  - Agent 集成
  - REST API

  Phase 4: 测试与优化（2-3 天）
  - 功能测试
  - 性能测试
  - 文档编写

  关键设计亮点

  1. 无侵入式设计：
    - 扩展现有 session_value 结构
    - 不影响现有快速路径性能
    - 向后兼容
  2. 可扩展架构：
    - 模块化协议检测器
    - 易于添加新协议
    - 支持自定义特征匹配
  3. 生产就绪：
    - 完整的错误处理
    - 性能监控指标
    - 配置热更新
  4. 参考实现：
    - 项目中已包含 NeuVector DPI 完整实现
    - 可参考其 20+ 协议检测器

  配置示例

  # 协议检测配置
  protocol_detection:
    enabled: true
    sampling_interval: 1        # 每个数据包检测
    max_payload_bytes: 128      # 最多检查 128 字节
    confidence_threshold: 70    # 最低置信度

  # 基于协议的策略
  policies:
    - name: "web-https-only"
      from:
        workload: "external"
      to:
        workload: "web-tier"
      protocol: HTTPS
      action: ALLOW

    - name: "deny-cleartext-http"
      from:
        workload: "external"
      to:
        workload: "web-tier"
      protocol: HTTP
      action: DENY

  技术挑战与解决方案

  文档中详细讨论了 5 个主要技术挑战：
  1. eBPF Verifier 复杂度限制
  2. Payload 访问安全性
  3. 加密流量检测
  4. 协议版本和变体
  5. 性能影响

  每个挑战都提供了具体的解决方案和缓解计划。