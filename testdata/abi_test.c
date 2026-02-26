// System MMIO port for printing integers
int* MMIO_DEC = 0xFF01;

void print_int(int val) {
    *MMIO_DEC = val;
}

// ---------------------------------------------------------
// TEST 1: Registers Only (Arguments in R4, R5, R6)
// ---------------------------------------------------------
int test_three_args(int a, int b, int c) {
    // Expected: 10 + 20 + 30 = 60
    return a + b + c;
}

// ---------------------------------------------------------
// TEST 2: Registers + Stack (R4, R5, R6, R7 + 2 on Stack)
// ---------------------------------------------------------
int test_six_args(int a, int b, int c, int d, int e, int f) {
    // Expected: 1 + 2 + 3 + 4 + 5 + 6 = 21
    return a + b + c + d + e + f;
}

// ---------------------------------------------------------
// TEST 3: Nested Call / Register Clobbering
// ---------------------------------------------------------
int multiply(int a, int b) {
    return a * b;
}

int test_nested(int a, int b, int c, int d) {
    // Expected: 2 + 15 + 6 + 7 = 30
    return a + b + c + d;
}


int main() {
    int res1 = test_three_args(10, 20, 30);
    print_int(res1);

    int res2 = test_six_args(1, 2, 3, 4, 5, 6);
    print_int(res2);

    // If the compiler doesn't evaluate and push everything to the
    // stack first, the call to `multiply(3, 5)` will clobber R4 and R5,
    // destroying the `2` we wanted to pass as argument `a`.
    int res3 = test_nested(2, multiply(3, 5), 6, 7);
    print_int(res3);

    return 0;
}
