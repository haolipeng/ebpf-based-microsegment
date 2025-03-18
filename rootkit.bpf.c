#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>

char LICENSE[] SEC("license") = "GPL";

//全局变量区
volatile int target_ppid = 0;

//宏定义
#define MAX_PID_LEN 10
const volatile int pid_to_hide_len = 0;
const volatile char pid_to_hide[MAX_PID_LEN] = {0};

// 映射表 存储dents 缓冲区的地址
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 8192);
    __type(key, u32);
    __type(value, u64);
} map_buffs SEC(".maps");

// 映射表，用于循环搜索数据
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 8192);
    __type(key, u32);
    __type(value, u32);
} map_bytes_read SEC(".maps");

// 映射表，存储程序的尾调用的
struct {
    __uint(type, BPF_MAP_TYPE_PROG_ARRAY);
    __uint(max_entries, 5);
    __type(key, u32);
    __type(value, u32);
}map_prog_array SEC(".maps");


//映射表，存储实际的地址
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 8192);
    __type(key, u32);
    __type(value, u64);
} map_to_patch SEC(".maps");

// RingBuffer to send events to user space
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024); // 256KB 的缓冲区
} rb SEC(".maps");

SEC("tracepoint/syscalls/sys_enter_getdents64")
int handle_getdents_enter(struct trace_event_raw_sys_enter *ctx) {	
    return 0;
}

SEC("tracepoint/syscalls/sys_exit_getdents64")
int handle_getdents_exit(struct trace_event_raw_sys_exit *ctx) {
    return 0;
}
