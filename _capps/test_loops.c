#include "../../lib/sys.c"
#include "../../lib/video.c"
#include "../../lib/stdio.c"



int main() {
    
    for (int i = 0; i < 5; i++) {
        print("Loop iteration: ");
        print_int(i);
        print("\\n");
    }

    return 0;
}
