#include <stdio.c>
#include <sys.c>

// Copied from lib/vfs.c but renamed globals to avoid conflicts
int* VFS_CMD_LOCAL    = 0xFF10;
int* VFS_NAME_LOCAL   = 0xFF11;
int* VFS_BUF_LOCAL    = 0xFF12;
int* VFS_SIZE_LOCAL   = 0xFF13;
int* VFS_STAT_LOCAL   = 0xFF14;
int* VFS_SIZE_H_LOCAL = 0xFF15;

int CMD_EXEC_WAIT_LOCAL = 8;

int vfs_read(int* filename, int* buffer) {
    *VFS_NAME_LOCAL = filename;
    *VFS_BUF_LOCAL  = buffer;
    *VFS_CMD_LOCAL  = 1;
    return *VFS_STAT_LOCAL;
}

int vfs_size(int* filename) {
    *VFS_NAME_LOCAL = filename;
    *VFS_CMD_LOCAL  = 3;
    if (*VFS_STAT_LOCAL != 0) {
        return -1;
    }
    return *VFS_SIZE_LOCAL;
}

int vfs_delete(int* filename) {
    *VFS_NAME_LOCAL = filename;
    *VFS_CMD_LOCAL  = 4;
    return *VFS_STAT_LOCAL;
}

int RECV_SLOT = -1;
int* INT_MASK = 0xFF09;
int* MMIO_SLOT_BASE = 0xFC00;

void isr() {
    int pending = *INT_MASK;

    if (RECV_SLOT != -1) {
        int mask = 1;
        for (int i = 0; i < RECV_SLOT; i++) {
            mask = mask * 2;
        }

        if ((pending & mask) != 0) {

            char buffer[256];
            char* filename = "INBOX.MSG";

            int size = vfs_size((int*)filename);

            if (size >= 0) {
                if (size < 255) {
                    int err = vfs_read((int*)filename, (int*)buffer);
                    if (err == 0) {
                        buffer[size] = 0;
                        print("New Message Received: ");
                        print(buffer);
                        print("\n");
                    } else {
                        print("Error reading inbox: ");
                        print_int(err);
                        print("\n");
                    }
                } else {
                    print("Error: Message too large\n");
                }
                vfs_delete((int*)filename);
            } else {
                print("Error: INBOX.MSG not found or invalid\n");
            }

            int* slot_addr = 0xFC00 + (RECV_SLOT * 16);
            *slot_addr = 1;

            // Only clear our interrupt
            *INT_MASK = mask;
        }
    }
}

int main() {
    print("Message Daemon Starting...\n");

    int* slot_ptr = find_peripheral("MSGRECV");
    if (slot_ptr == 0) {
        print("Error: Message Receiver Peripheral not found!\n");
        return 0;
    }

    int address = (int)slot_ptr;
    int offset = address - 0xFC00;
    RECV_SLOT = offset / 16;

    if (RECV_SLOT < 0) { RECV_SLOT = 0; }
    if (RECV_SLOT > 15) { RECV_SLOT = 15; }

    print("Found MSGRECV at slot: ");
    print_int(RECV_SLOT);
    print("\n");

    enable_interrupts();
    print("Interrupts enabled. Waiting for messages...\n");

    while (1) {
        wait_for_interrupt();
    }

    return 0;
}
