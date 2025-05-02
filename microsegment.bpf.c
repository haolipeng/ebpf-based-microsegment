#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_tracing.h>

char LICENSE[] SEC("license") = "GPL";

SEC("raw_tracepoint/sys_enter")
int enter_fchmodat(struct bpf_raw_tracepoint_args *ctx) {	
    struct pt_regs* regs;
    regs = (struct pt_regs*)ctx->args[0];

    //int fchmodat(int dirfd, const char* pathname, mode_t mode, int flags);
    char pathname[256];
    char* pathname_ptr = (char*)PT_REGS_PARM2_CORE(regs);
    bpf_core_read_user_str(pathname, sizeof(pathname), pathname_ptr);

    char fmt[] = "fchmodat %s\n";
    bpf_trace_printk(fmt, sizeof(fmt), &pathname);
    return 0;
}

// 映射表，存储程序的尾调用的
struct {
    __uint(type, BPF_MAP_TYPE_PROG_ARRAY);
    __uint(max_entries, 1024);
    __uint(key_size, sizeof(u32));
    __uint(value_size, sizeof(u32));
    __array(values,int (void*));
}tail_jump_map SEC(".maps") = {
    .values = {
        [268] = (void*)&enter_fchmodat,
    },
};

SEC("raw_tracepoint/sys_enter")
int raw_tracepoint__sys_enter(struct bpf_raw_tracepoint_args *ctx) {
    //call another ebpf program
    u32 syscall_id = 268;
    bpf_tail_call(ctx,&tail_jump_map,syscall_id);

    char fmt[] = "no bpf program for syscall %d\n";
    bpf_trace_printk(fmt, sizeof(fmt), syscall_id);

    return 0;
}
