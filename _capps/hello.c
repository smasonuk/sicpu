// File: cmd/desktop/apps/hello.c
#include "../../../lib/sys.c"
#include "../../../lib/stdio.c"

int main() {
    print("  ---> [hello.bin] Spawned process is now running!\n");
    print("  ---> [hello.bin] Halting to trigger hypervisor return...\n");
    return 0; // This translates to HLT, which should trigger the restore
}
