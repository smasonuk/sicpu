#include "../lib/stdio.c"

int* MSG_CMD = 0xFC00; //writing into slot 1
int* MSG_TO  = 0xFC02;
int* MSG_BODY = 0xFC04;
int* MSG_LEN = 0xFC06;

void send_msg(byte* to, byte* body, int len) {
    *MSG_TO = to;
    *MSG_BODY = body;
    *MSG_LEN = len;
    *MSG_CMD = 1;
}

int main() {
    byte* target = "Central Command";

    byte* payload = "Ground control to major Tom"; // This will set the first 4 bytes of payload to 'A', 'B', 'C', 'D'
    int len = strlen(payload);

    send_msg(target, payload, len);

    print("Message sent!\n");
    return 0;
}
