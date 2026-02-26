#include "../lib/sys.c"
#include "../lib/stdio.c"
#include "../lib/vfs.c"

int main() {
    print("VFS Test Start\n");

    byte buffer[20];
    buffer[0] = 72;  // 'H'
    buffer[1] = 69;  // 'E'
    buffer[2] = 76;  // 'L'
    buffer[3] = 76;  // 'L'
    buffer[4] = 79;  // 'O'
    buffer[5] = 0;   // Null terminator
    
    byte* filename = "TEST.TXT";

    print("Buffer before save: ");
    print(buffer);
    print("\n");

    print("Saving...\n");
    // Save 6 bytes (hardware expects bytes despite spec saying words)
    int err = vfs_write(filename, buffer, 6);

    if (err != 0) {
        print("Save Failed. Error: ");
        print_int(err);
        print("\n");
    } else {
        print("Save Success\n");
    }

    print("Clearing buffer...\n");
    // Clear 10 words (20 bytes)
    memset(buffer, 10, 0);

    print("Buffer after clear: ");
    print(buffer);
    print("\n");

    print("Loading...\n");
    err = vfs_read(filename, buffer);

    if (err != 0) {
        print("Load Failed. Error: ");
        print_int(err);
        print("\n");
    } else {
        print("Load Success\n");
    }

    print("Buffer after load: ");
    print(buffer);
    print("\n");

    print("VFS Test Done\n");
    return 0;
}
