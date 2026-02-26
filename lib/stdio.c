// Standard I/O Hardware Ports
int* STDOUT_PORT = 0xFF00;
int* MMIO_DEC = 0xFF01;

// Prints a standard null-terminated byte string
void print(byte* str) {
    while (*str != 0) {
        *STDOUT_PORT = *str;
        str++;
    }
}

// Prints an integer as decimal
void print_int(int val) {
    *MMIO_DEC = val;
}

// Calculates the length of a null-terminated string
int strlen(byte* str) {
    int len = 0;
    while (*str != 0) {
        len++;
        str++;
    }
    return len;
}

// Copies the source string to the destination buffer
void strcpy(byte* dest, byte* src) {
    while (*src != 0) {
        *dest = *src;
        dest++;
        src++;
    }
    *dest = 0;
}

// Compares two strings. Returns 0 if equal, <0 if s1 < s2, >0 if s1 > s2
int strcmp(byte* s1, byte* s2) {
    while (*s1 != 0 && *s2 != 0) {
        if (*s1 != *s2) {
            return *s1 - *s2;
        }
        s1++;
        s2++;
    }
    return *s1 - *s2;
}

// Concatenates src to the end of dest
void strcat(byte* dest, byte* src) {
    // Find end of dest
    while (*dest != 0) {
        dest++;
    }
    // Copy src
    while (*src != 0) {
        *dest = *src;
        dest++;
        src++;
    }
    *dest = 0;
}

// Reverses a string in place
void reverse(byte* s) {
    int len = strlen(s);
    if (len == 0) { return; }

    byte* start = s;
    byte* end = s;
    
    // Move end to the last character
    int i = 0;
    while (i < len - 1) {
        end++;
        i++;
    }

    while (start < end) {
        byte temp = *start;
        *start = *end;
        *end = temp;
        start++;
        end--;
    }
}

// Converts an integer to a null-terminated string
void itoa(int n, byte* s) {
    int i = 0;
    int sign = n;
    if (sign < 0) {
        n = -n;
    }

    // Handle 0 explicitly
    if (n == 0) {
        *s = 48; // '0'
        s++;
        *s = 0;
        return;
    }

    byte* start = s;

    while (n > 0) {
        int rem = n % 10;
        *s = rem + 48; // '0'
        s++;
        n = n / 10;
    }

    if (sign < 0) {
        *s = 45; // '-'
        s++;
    }
    *s = 0;

    reverse(start);
}


void print_array(int* arr, int len) {
    print("[");
    for (int i = 0; i < len; i++) {
        print_int(arr[i]);
        if (i < len - 1) {
            print(", ");
        }
    }
    print("]");
}