#include "../../lib/stdio.c"

int main() {
    print("Testing stdio strings...\\n");

    // Test strlen
    byte* s1 = "Hello";
    int len = strlen(s1);
    print("strlen('Hello'): ");
    print_int(len); // Expected: 5
    print("\\n");
    
    // Test strcpy
    byte buf[20];
    
    // Initialize buffer with 'A'
    buf[0] = 65; // 'A'
    buf[1] = 0;
    
    print("Before strcpy: ");
    print(buf);
    print("\\n");
    
    strcpy(buf, s1);
    print("After strcpy: ");
    print(buf);
    print("\\n");
    
    // Test strcmp
    byte* s2 = "Hello";
    byte* s3 = "World";
    int cmp = strcmp(s1, s2);
    print("strcmp('Hello', 'Hello'): ");
    print_int(cmp); // Expected: 0
    print("\\n");
    
    cmp = strcmp(s1, s3);
    print("strcmp('Hello', 'World'): ");
    // 'H' is 72, 'W' is 87. 72 - 87 = -15.
    print_int(cmp); 
    print("\\n");

    // Test reverse
    byte buf2[10];
    buf2[0] = 49; // '1'
    buf2[1] = 50; // '2'
    buf2[2] = 51; // '3'
    buf2[3] = 0;
    
    print("Before reverse: ");
    print(buf2);
    print("\\n");
    
    reverse(buf2);
    
    print("After reverse: ");
    print(buf2); // Expected: "321"
    print("\\n");
    
    // Test itoa
    print("itoa(12345): ");
    byte numBuf[12];
    itoa(12345, numBuf);
    print(numBuf);
    print("\\n");
    
    print("itoa(-99): ");
    itoa(-99, numBuf);
    print(numBuf);
    print("\\n");
    
    // Test strcat
    byte buf3[30];
    // Copy "Hello"
    strcpy(buf3, s1);
    // Cat " World"
    byte* sWorld = " World";
    strcat(buf3, sWorld);
    print("strcat: ");
    print(buf3); // "Hello World"
    print("\\n");
    
    return 0;
}
