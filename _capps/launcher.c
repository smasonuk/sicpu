#include "../lib/sys.c"
#include "../lib/stdio.c"
#include "../lib/vfs.c"

int main() {
    print("[launcher.bin] System Booting...\n");
    print("[launcher.bin] Preparing to execute hello.bin...\n\n");

    // The new command!
    int err = vfs_exec_wait("hello.bin");

    // We only get here if the hypervisor successfully restored our RAM and Registers
    print("\n[launcher.bin] Hypervisor restored context!\n");

    if (err != 0) {
        print("[launcher.bin] ERROR: Exec failed with code: ");
        print_int(err);
        print("\n");
    } else {
        print("[launcher.bin] SUCCESS: Execution completed normally.\n");
    }

    print("[launcher.bin] Shutting down.\n");
    return 0;
}
