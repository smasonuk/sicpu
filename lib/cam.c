// Triggers the camera, then transmits the raw RGB332 frame to Earth
// via the MessageSender peripheral on the message bus.

// Slot 0: Message Sender (0xFC00)
#define MSGSNDR_CMD  0xFC00
#define MSGSNDR_TO   0xFC02
#define MSGSNDR_BODY 0xFC04
#define MSGSNDR_LEN  0xFC06

// Slot 1: Camera (0xFC10)
#define CAMERA_CMD   0xFC10
#define CAMERA_BUF   0xFC12
#define CAMERA_W     0xFC14
#define CAMERA_H     0xFC16

// Image buffer: 128x128 = 16384 bytes, placed safely in mid-RAM (away from
// code at 0x0010 and stack at 0xFFFE).
#define IMAGE_BUFFER 0x4000
#define IMAGE_SIZE   16384

// TODO: find the peripheral

void take_picture_and_send(byte* send_to_address) {
    // 1. Configure the camera buffer, dimensions, then trigger capture.
    int* cam_buf = (int*)CAMERA_BUF;
    *cam_buf = IMAGE_BUFFER;

    int* cam_w = (int*)CAMERA_W;
    *cam_w = 128;

    int* cam_h = (int*)CAMERA_H;
    *cam_h = 128;

    int* cam_cmd = (int*)CAMERA_CMD;
    *cam_cmd = 1;

    int* send_to = (int*)MSGSNDR_TO;
    *send_to = send_to_address;

    int* send_body = (int*)MSGSNDR_BODY;
    *send_body = IMAGE_BUFFER;

    int* send_len = (int*)MSGSNDR_LEN;
    *send_len = IMAGE_SIZE;

    int* send_cmd = (int*)MSGSNDR_CMD;
    *send_cmd = 1;
}