#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include "common_um.h"
#include "microsegment.skel.h"
#include <bpf/libbpf.h>

int main(int argc, char *argv[]) {
    struct microsegment_bpf *skel;
    int err;

    /* Setup common tasks*/
    if (!setup()) {
        fprintf(stderr, "Failed to do common setup\n");
        return 1;
    };

    /* Open BPF application */
    skel = microsegment_bpf__open();
    if (!skel) {
        fprintf(stderr, "Failed to open BPF skeleton\n");
        return 1;
    }

    /* Load & verify BPF programs */
    err = microsegment_bpf__load(skel);
    if (err) {
        fprintf(stderr, "Failed to load and verify BPF skeleton\n");
        goto cleanup;
    }

    /*
    int index = 1;
    int prog_fd = bpf_program__fd(skel->progs.raw_tracepoint__sys_enter);
    int ret = bpf_map__update_elem();
    */

    /* Attach tracepoint handler */
    err = microsegment_bpf__attach(skel);
    if (err) {
        fprintf(stderr, "Failed to attach BPF skeleton\n");
        goto cleanup;
    }

cleanup:
    microsegment_bpf__destroy(skel);
    return err < 0 ? -err : 0;
}
