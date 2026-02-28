#define MSGSNDR_TO_OFFSET   1
#define MSGSNDR_BODY_OFFSET 2
#define MSGSNDR_LEN_OFFSET  3
#define MSGSNDR_SEND        1

int send_message(char* send_to_address, int image_buffer_address, int image_size) {
    int* sender = find_peripheral("MSGSNDR");
    if (sender == 0) {
        print("Camera not found!\n");
        return 1;
    }

    //send message
    int* send_to = (int*)(sender + MSGSNDR_TO_OFFSET);
    *send_to = send_to_address;

    int* send_body = (int*)(sender + MSGSNDR_BODY_OFFSET);
    *send_body = image_buffer_address;

    int* send_len = (int*)(sender + MSGSNDR_LEN_OFFSET);
    *send_len = image_size;

    int* send_cmd = (int*)sender;
    *send_cmd = MSGSNDR_SEND; 
}