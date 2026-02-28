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


    // from: local
    // to: navigation@local

    // from: navigation@local
    // to: @local

    // from: earth
    // to navifation@probe_id
    // message: move_to("x,y,z", "speed")


    if (RECV_SLOT != -1) { // this is the slot where the message reciever is attatched
        int mask = 1;
        for (int i = 0; i < RECV_SLOT; i++) {
            mask = mask * 2;
        }

        if ((pending & mask) != 0) {

            char buffer[256];
            char sender_buffer[256];
            char* filename = "INBOX.MSG";
            char* sender_filename = "SENDER.MSG";

            int size = vfs_size((int*)filename);
            int sender_size = vfs_size((int*)sender_filename);

            if (size >= 0 && sender_size >= 0) {
                if (size < 255 && sender_size < 255) {
                    int err_sender = vfs_read((int*)sender_filename, (int*)sender_buffer);
                    int err_msg = vfs_read((int*)filename, (int*)buffer);
                    
                    if (err_sender == 0 && err_msg == 0) {
                        sender_buffer[sender_size] = 0;
                        buffer[size] = 0;
                        
                        print("Message Received from ");
                        print(sender_buffer);
                        print(": ");
                        print(buffer);
                        print("\n");
                    } else {
                        print("Error reading messages. Sender err: ");
                        print_int(err_sender);
                        print(", Msg err: ");
                        print_int(err_msg);
                        print("\n");
                    }
                } else {
                    print("Error: Message or Sender too large\n");
                }
                vfs_delete((int*)filename);
                vfs_delete((int*)sender_filename);
            } else {
                print("Error: INBOX.MSG or SENDER.MSG not found or invalid\n");
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
