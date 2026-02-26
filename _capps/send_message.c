#include "../lib/stdio.c"
#include "../lib/sys.c"

int main() {
    int slot = find_peripheral("MSGSNDR");

    byte* target = "Central Command";

    byte* payload = "Ground control to major Tom"; // This will set the first 4 bytes of payload to 'A', 'B', 'C', 'D'
    int len = strlen(payload);

    send_msg(slot, target, payload, len);

    print("Message sent!\n");
    return 0;
}
