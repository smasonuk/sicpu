// VFS MMIO Hardware Ports
int* VFS_CMD    = 0xFF10; // Command trigger: 1=Read, 2=Write, 3=Size, 4=Delete, 5=List, 6=FreeSpace, 7=GetMeta
int* VFS_NAME   = 0xFF11; // Pointer to null-terminated filename string
int* VFS_BUF    = 0xFF12; // Pointer to data buffer
int* VFS_SIZE   = 0xFF13; // Size in bytes (16-bit)
int* VFS_STAT   = 0xFF14; // Status code: 0=Success, 1=NotFound, 2=Full, 3=InvalidName, 4=OutOfBounds, 5=DirEnd
int* VFS_SIZE_H = 0xFF15; // High word for free space calculation

int CMD_EXEC_WAIT = 8;

// Writes 'length' bytes from 'buffer' to 'filename'.
// Returns 0 on success, or an error code (1-4).
int vfs_write(int* filename, int* buffer, int length) {
    *VFS_NAME = filename;
    *VFS_BUF  = buffer;
    *VFS_SIZE = length;

    *VFS_CMD  = 2; // Trigger Write Command

    return *VFS_STAT;
}

// Pauses the current program, loads and runs 'filename', and resumes
// when the loaded program halts. Returns 0 on success.
int vfs_exec_wait(int* filename) {
    *VFS_NAME = filename;
    *VFS_CMD  = CMD_EXEC_WAIT;
    return *VFS_STAT;
}

// Reads 'filename' into 'buffer'.
// Ensure 'buffer' is large enough to hold the file!
// Returns 0 on success.
int vfs_read(int* filename, int* buffer) {
    *VFS_NAME = filename;
    *VFS_BUF  = buffer;

    *VFS_CMD  = 1; // Trigger Read Command

    return *VFS_STAT;
}

// Gets the size of a file in words.
// Returns the size, or -1 if the file doesn't exist.
int vfs_size_calc(int* filename) {
    *VFS_NAME = filename;
    *VFS_CMD  = 3; // Trigger Size Command

    if (*VFS_STAT != 0) {
        return -1; // Return -1 on error
    }
    return *VFS_SIZE;
}

// Deletes the specified file.
// Returns 0 on success.
int vfs_delete(int* filename) {
    *VFS_NAME = filename;
    *VFS_CMD  = 4; // Trigger Delete Command

    return *VFS_STAT;
}
