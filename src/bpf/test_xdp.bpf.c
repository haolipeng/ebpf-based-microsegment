// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* 最小的测试 XDP 程序,用于检测网卡驱动是否支持 Native XDP */

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

/* 最简单的 XDP 程序 - 只返回 XDP_PASS,不做任何处理 */
SEC("xdp")
int xdp_test_prog(struct xdp_md *ctx)
{
	/* 直接通过所有数据包 */
	return XDP_PASS;
}
