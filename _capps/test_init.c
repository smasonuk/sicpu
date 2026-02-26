#include "../../lib/sys.c"
#include "../../lib/video.c"
#include "../../lib/stdio.c"

int g[3] = {10, 20, 30};


int main() {

    g[1] = -123;

    print_array(g, 3);
    print("\\n");

    int i = -1 + 5;

    print_int(i);



    return g[1];
}