#define CAMERA_BUF_OFFSET   0x0002
#define CAMERA_W_OFFSET     0x0004
#define CAMERA_H_OFFSET     0x0006


// Image buffer: 128x128 = 16384 bytes, placed safely in mid-RAM (away from
// code at 0x0010 and stack at 0xFFFE).
#define IMAGE_BUFFER 0x4000
#define IMAGE_SIZE   16384

int takepicture(int buffer_start) {
    int CAM_CAPTURE_COMMAND = 1;

    int* cam = find_peripheral("CAMERA");
    if (cam == 0) {
        print("Camera not found!\n");
        return 1;
    }

    //capture image
    int* cam_buf = (int*)(cam + CAMERA_BUF_OFFSET);
    *cam_buf = buffer_start;
    int* cam_w = (int*)(cam + CAMERA_W_OFFSET);
    *cam_w = 128;
    int* cam_h = (int*)(cam + CAMERA_H_OFFSET);
    *cam_h = 128;

    // tell the camera to capture the image
    *cam = CAM_CAPTURE_COMMAND; 
}