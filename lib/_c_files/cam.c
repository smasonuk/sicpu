#define CAMERA_BUF_OFFSET   1
#define CAMERA_W_OFFSET     2
#define CAMERA_H_OFFSET     3


int takepicture(int buffer_start) {
    int CAM_CAPTURE_COMMAND = 1;

    int* cam = find_peripheral("CAMERA");
    if (cam == 0) {
        print("Camera not found!\n");
        return 1;
    }

    print_int(buffer_start);
    print("\n" );

    //capture image
    int* cam_buf = (int*)(cam + CAMERA_BUF_OFFSET);
    *cam_buf = buffer_start;
    int* cam_w = (int*)(cam + CAMERA_W_OFFSET);
    *cam_w = 128;
    int* cam_h = (int*)(cam + CAMERA_H_OFFSET);
    *cam_h = 128;

    print("1!\n");
    // tell the camera to capture the image
    *cam = CAM_CAPTURE_COMMAND; 
    print("2!\n");
}