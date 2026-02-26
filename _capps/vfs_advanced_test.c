// vfs_advanced_test.csrc

// MMIO Constants
#define VFS_CMD 0xFF10
#define VFS_NAME 0xFF11
#define VFS_BUF 0xFF12
#define VFS_SIZE 0xFF13
#define VFS_STAT 0xFF14
#define VFS_SIZE_H 0xFF15

// VFS Commands
#define CMD_DELETE 4
#define CMD_LIST 5
#define CMD_FREESPACE 6
#define CMD_GETMETA 7

// Error Codes
#define ERR_SUCCESS 0
#define ERR_NOTFOUND 1
#define ERR_DISKFULL 2
#define ERR_INVALIDNAME 3
#define ERR_OUTOFBOUNDS 4
#define ERR_DIREND 5

// Helper to print strings
void print_str(char* s) {
  int i = 0;
  while (s[i] != 0) {
    *0xFF00 = s[i];
    i++;
  }
}

// Helper to print integers
void print_int(int n) {
  *0xFF01 = n;
}

void main() {
  print_str("VFS Advanced Test\n");

  // 1. Check Free Space
  *0xFF10 = CMD_FREESPACE;
  int low = *VFS_SIZE;
  int high = *VFS_SIZE_H;
  print_str("Free Space (Low): ");
  print_int(low);
  print_str("Free Space (High): ");
  print_int(high);
  // Reconstruct full size (simulated as we can't print 32-bit easily if int is 16-bit)
  // Assuming int is 16-bit in this architecture.
  print_str("\n");

  // 2. Create a file
  char* filename = "testfile.txt";
  char* data = "Hello VFS!";
  int len = 10;

  *VFS_NAME = filename;
  *VFS_BUF = data;
  *VFS_SIZE = len;
  *0xFF10 = 2; // CMD_WRITE

  if (*VFS_STAT == ERR_SUCCESS) {
    print_str("File created: ");
    print_str(filename);
    print_str("\n");
  } else {
    print_str("Failed to create file.\n");
  }

  // 3. Get Metadata
  int meta_buf[12];
  *VFS_NAME = filename;
  *VFS_BUF = meta_buf;
  *0xFF10 = CMD_GETMETA;

  if (*VFS_STAT == ERR_SUCCESS) {
    print_str("Creation Date: ");
    print_int(meta_buf[0]); // Year
    print_str("-");
    print_int(meta_buf[1]); // Month
    print_str("\n");
  } else {
    print_str("Failed to get metadata.\n");
  }

  // 4. List Files
  print_str("Listing Files:\n");
  char name_buf[20];
  *VFS_BUF = name_buf;

  // Reset list iteration by calling list once?
  // The first call initiates it.

  while (1) {
    *0xFF10 = CMD_LIST;
    if (*VFS_STAT == ERR_DIREND) {
      break;
    }
    print_str("- ");
    print_str(name_buf);
    print_str("\n");
  }

  // 5. Delete File
  *VFS_NAME = filename;
  *0xFF10 = CMD_DELETE;

  if (*VFS_STAT == ERR_SUCCESS) {
    print_str("File deleted.\n");
  } else {
    print_str("Failed to delete file.\n");
  }

  // Verify deletion
  *VFS_NAME = filename;
  *0xFF10 = 3; // CMD_SIZE
  if (*VFS_STAT == ERR_NOTFOUND) {
    print_str("Verification: File not found (correct).\n");
  } else {
    print_str("Verification: File still exists!\n");
  }

}
