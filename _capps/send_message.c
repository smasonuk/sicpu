#include "../lib/stdio.c"

int* MSG_CMD = 0xFC00;
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
    byte payload[4];
    payload[0] = 0xDE;
    payload[1] = 0xAD;
    payload[2] = 0xBE;
    payload[3] = 0xEF;

    send_msg(target, payload, 4);

    print("Message sent!\n");
    return 0;
}
