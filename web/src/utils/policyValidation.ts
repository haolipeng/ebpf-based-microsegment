import type { Policy } from '../types/policy'

/**
 * 检查两个 IP 地址/CIDR 范围是否重叠
 */
function ipRangesOverlap(ip1: string, ip2: string): boolean {
  // 解析 IP 和 CIDR
  const parseIpCidr = (ipCidr: string): { ip: number; mask: number } => {
    const parts = ipCidr.split('/')
    const ipParts = parts[0].split('.').map(Number)
    const ip = (ipParts[0] << 24) | (ipParts[1] << 16) | (ipParts[2] << 8) | ipParts[3]
    const prefix = parts.length > 1 ? parseInt(parts[1]) : 32
    const mask = prefix === 0 ? 0 : ~((1 << (32 - prefix)) - 1)
    return { ip: ip >>> 0, mask: mask >>> 0 }
  }

  const range1 = parseIpCidr(ip1)
  const range2 = parseIpCidr(ip2)

  // 检查网络地址是否相同
  const network1 = range1.ip & range1.mask
  const network2 = range2.ip & range2.mask

  // 检查是否重叠
  if (range1.mask === range2.mask) {
    return network1 === network2
  }

  // 检查一个范围是否包含另一个
  const smallerMask = range1.mask > range2.mask ? range1.mask : range2.mask
  const network1Masked = range1.ip & smallerMask
  const network2Masked = range2.ip & smallerMask

  return network1Masked === network2Masked
}

/**
 * 检查端口是否匹配
 */
function portsMatch(port1: number, port2: number): boolean {
  // 端口 0 表示任意端口
  if (port1 === 0 || port2 === 0) {
    return true
  }
  return port1 === port2
}

/**
 * 检查协议是否匹配
 */
function protocolsMatch(protocol1: string, protocol2: string): boolean {
  if (protocol1 === 'any' || protocol2 === 'any') {
    return true
  }
  return protocol1 === protocol2
}

/**
 * 检查两个策略是否冲突
 */
export function policiesConflict(policy1: Policy, policy2: Policy): boolean {
  // 跳过已禁用的策略
  if (policy1.enabled === false || policy2.enabled === false) {
    return false
  }

  // 检查 IP 范围是否重叠
  const srcIpOverlap = ipRangesOverlap(policy1.srcIp, policy2.srcIp)
  const dstIpOverlap = ipRangesOverlap(policy1.dstIp, policy2.dstIp)

  if (!srcIpOverlap || !dstIpOverlap) {
    return false
  }

  // 检查端口是否匹配
  const srcPortMatch = portsMatch(policy1.srcPort, policy2.srcPort)
  const dstPortMatch = portsMatch(policy1.dstPort, policy2.dstPort)

  if (!srcPortMatch || !dstPortMatch) {
    return false
  }

  // 检查协议是否匹配
  const protocolMatch = protocolsMatch(policy1.protocol, policy2.protocol)

  if (!protocolMatch) {
    return false
  }

  // 如果所有条件都匹配，则存在冲突
  return true
}

/**
 * 验证策略并返回冲突列表
 */
export function validatePolicy(
  policy: Omit<Policy, 'ruleId'> | Policy,
  existingPolicies: Policy[]
): {
  valid: boolean
  conflicts: Policy[]
  warnings: string[]
} {
  const conflicts: Policy[] = []
  const warnings: string[] = []

  // 检查与现有策略的冲突
  for (const existingPolicy of existingPolicies) {
    // 跳过同一策略（编辑时）
    if ('ruleId' in policy && policy.ruleId === existingPolicy.ruleId) {
      continue
    }

    if (policiesConflict(policy as Policy, existingPolicy)) {
      conflicts.push(existingPolicy)
    }
  }

  // 检查优先级警告
  if ('priority' in policy && policy.priority !== undefined) {
    if (policy.priority < 10) {
      warnings.push('Priority is very high (< 10), this policy will be evaluated first')
    } else if (policy.priority > 900) {
      warnings.push('Priority is very low (> 900), this policy may rarely be evaluated')
    }
  }

  // 检查过于宽泛的规则
  if (policy.srcIp.includes('/8') || policy.dstIp.includes('/8')) {
    warnings.push('Very broad IP range (/8), this policy will affect many connections')
  }

  if (policy.protocol === 'any' && policy.srcPort === 0 && policy.dstPort === 0) {
    warnings.push('Policy matches all protocols and ports, consider being more specific')
  }

  // 检查 deny 规则警告
  if (policy.action === 'deny' && conflicts.length > 0) {
    warnings.push('This deny rule conflicts with existing policies, make sure the priority is correct')
  }

  return {
    valid: conflicts.length === 0,
    conflicts,
    warnings,
  }
}

/**
 * 批量验证策略
 */
export function validatePolicies(policies: Policy[]): {
  valid: boolean
  conflictPairs: Array<{ policy1: Policy; policy2: Policy }>
} {
  const conflictPairs: Array<{ policy1: Policy; policy2: Policy }> = []

  for (let i = 0; i < policies.length; i++) {
    for (let j = i + 1; j < policies.length; j++) {
      if (policiesConflict(policies[i], policies[j])) {
        conflictPairs.push({
          policy1: policies[i],
          policy2: policies[j],
        })
      }
    }
  }

  return {
    valid: conflictPairs.length === 0,
    conflictPairs,
  }
}
