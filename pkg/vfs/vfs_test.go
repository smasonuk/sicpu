package vfs

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestVirtualDisk_Write(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		data         []byte
		initialUsed  int
		expectError  bool
		expectedUsed int
	}{
		{
			name:         "Valid write",
			filename:     "test.txt",
			data:         []byte{1, 2, 3},
			initialUsed:  0,
			expectError:  false,
			expectedUsed: 3,
		},
		{
			name:         "Invalid filename special chars",
			filename:     "test!.txt",
			data:         []byte{1},
			initialUsed:  0,
			expectError:  true,
			expectedUsed: 0,
		},
		{
			name:         "Invalid filename too long",
			filename:     "verylongfilename.txt",
			data:         []byte{1},
			initialUsed:  0,
			expectError:  true,
			expectedUsed: 0,
		},
		{
			name:         "Invalid filename path traversal",
			filename:     "../passwd",
			data:         []byte{1},
			initialUsed:  0,
			expectError:  true,
			expectedUsed: 0,
		},
		{
			name:         "Quota exceeded",
			filename:     "bigfile.bin",
			data:         make([]byte, MaxDiskBytes+1),
			initialUsed:  0,
			expectError:  true,
			expectedUsed: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vd := NewVirtualDisk()
			vd.UsedBytes = tt.initialUsed
			err := vd.Write(tt.filename, tt.data)

			if (err != nil) != tt.expectError {
				t.Errorf("Write() error = %v, expectError %v", err, tt.expectError)
			}

			if !tt.expectError {
				if vd.UsedBytes != tt.expectedUsed {
					t.Errorf("UsedBytes = %d, expected %d", vd.UsedBytes, tt.expectedUsed)
				}
				stored, ok := vd.Files[tt.filename]
				if !ok {
					t.Errorf("File %s not found in map", tt.filename)
				}
				if !reflect.DeepEqual(stored.Data, tt.data) {
					t.Errorf("Stored data = %v, expected %v", stored.Data, tt.data)
				}
				if stored.Created.IsZero() || stored.Modified.IsZero() {
					t.Errorf("Timestamps not set: Created=%v, Modified=%v", stored.Created, stored.Modified)
				}
			}
		})
	}
}

func TestVirtualDisk_Read(t *testing.T) {
	vd := NewVirtualDisk()
	filename := "test.txt"
	data := []byte{10, 20, 30}
	vd.Write(filename, data)

	tests := []struct {
		name        string
		filename    string
		expectError bool
		expectData  []byte
	}{
		{
			name:        "Read existing file",
			filename:    "test.txt",
			expectError: false,
			expectData:  data,
		},
		{
			name:        "Read non-existent file",
			filename:    "missing.txt",
			expectError: true,
			expectData:  nil,
		},
		{
			name:        "Read invalid filename",
			filename:    "../passwd",
			expectError: true,
			expectData:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := vd.Read(tt.filename)
			if (err != nil) != tt.expectError {
				t.Errorf("Read() error = %v, expectError %v", err, tt.expectError)
			}
			if !tt.expectError && !reflect.DeepEqual(got, tt.expectData) {
				t.Errorf("Read() got = %v, want %v", got, tt.expectData)
			}
		})
	}
}

func TestVirtualDisk_Size(t *testing.T) {
	vd := NewVirtualDisk()
	filename := "test.txt"
	data := []byte{10, 20, 30}
	vd.Write(filename, data)

	tests := []struct {
		name        string
		filename    string
		expectError bool
		expectSize  int
	}{
		{
			name:        "Size existing file",
			filename:    "test.txt",
			expectError: false,
			expectSize:  3,
		},
		{
			name:        "Size non-existent file",
			filename:    "missing.txt",
			expectError: true,
			expectSize:  0,
		},
		{
			name:        "Size invalid filename",
			filename:    "../passwd",
			expectError: true,
			expectSize:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, err := vd.Size(tt.filename)
			if (err != nil) != tt.expectError {
				t.Errorf("Size() error = %v, expectError %v", err, tt.expectError)
			}
			if !tt.expectError && size != tt.expectSize {
				t.Errorf("Size() size = %d, want %d", size, tt.expectSize)
			}
		})
	}
}

func TestVirtualDisk_UpdateFileSize(t *testing.T) {
	vd := NewVirtualDisk()
	filename := "update.txt"

	data1 := []byte{1, 2, 3, 4, 5}
	err := vd.Write(filename, data1)
	if err != nil {
		t.Fatalf("Initial Write failed: %v", err)
	}
	if vd.UsedBytes != 5 {
		t.Errorf("UsedBytes after initial write = %d, expected 5", vd.UsedBytes)
	}

	entry1, _ := vd.Files[filename]
	created1 := entry1.Created

	time.Sleep(1 * time.Millisecond)

	data2 := []byte{1, 2, 3, 4, 5, 6, 7}
	err = vd.Write(filename, data2)
	if err != nil {
		t.Fatalf("Update (larger) failed: %v", err)
	}
	if vd.UsedBytes != 7 {
		t.Errorf("UsedBytes after larger update = %d, expected 7", vd.UsedBytes)
	}

	entry2, _ := vd.Files[filename]
	if !entry2.Created.Equal(created1) {
		t.Error("Created time should not change on update")
	}
	if !entry2.Modified.After(entry2.Created) {
		t.Error("Modified time should be after Created time after update")
	}

	data3 := []byte{1, 2}
	err = vd.Write(filename, data3)
	if err != nil {
		t.Fatalf("Update (smaller) failed: %v", err)
	}
	if vd.UsedBytes != 2 {
		t.Errorf("UsedBytes after smaller update = %d, expected 2", vd.UsedBytes)
	}
}

func TestVirtualDisk_DeepCopy(t *testing.T) {
	vd := NewVirtualDisk()
	filename := "mutable.txt"
	data := []byte{1, 2, 3}

	err := vd.Write(filename, data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	data[0] = 99

	readData, err := vd.Read(filename)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if readData[0] == 99 {
		t.Error("Write did not perform a deep copy; mutation of source affected stored data")
	}
}

func TestVirtualDisk_QuotaExact(t *testing.T) {
	vd := NewVirtualDisk()

	filename1 := "file1.bin"
	data1 := make([]byte, MaxDiskBytes-1)
	err := vd.Write(filename1, data1)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	filename2 := "file2.bin"
	data2 := []byte{1, 2}
	err = vd.Write(filename2, data2)
	if err == nil {
		t.Error("Expected quota error, got nil")
	}

	data3 := []byte{1}
	err = vd.Write(filename2, data3)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	if vd.UsedBytes != MaxDiskBytes {
		t.Errorf("UsedBytes = %d, expected %d", vd.UsedBytes, MaxDiskBytes)
	}
}

func TestVirtualDisk_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vfs_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	vd := NewVirtualDisk()

	vd.Write("file1.txt", []byte{'a'})
	if !vd.DirtyFiles["file1.txt"] {
		t.Error("file1.txt should be dirty")
	}
	if !vd.Dirty {
		t.Error("Disk should be dirty")
	}

	vd.Write("file2.txt", []byte{'b'})
	if !vd.DirtyFiles["file2.txt"] {
		t.Error("file2.txt should be dirty")
	}

	err = vd.PersistTo(tempDir)
	if err != nil {
		t.Fatalf("PersistTo failed: %v", err)
	}

	if len(vd.DirtyFiles) != 0 {
		t.Errorf("DirtyFiles should be empty, got %d", len(vd.DirtyFiles))
	}
	if vd.Dirty {
		t.Error("Disk should not be dirty after persist")
	}

	if _, err := os.Stat(filepath.Join(tempDir, "file1.txt")); os.IsNotExist(err) {
		t.Error("file1.txt not persisted")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "file2.txt")); os.IsNotExist(err) {
		t.Error("file2.txt not persisted")
	}

	vd.Write("file1.txt", []byte{'c'})
	if !vd.DirtyFiles["file1.txt"] {
		t.Error("file1.txt should be dirty")
	}
	if vd.DirtyFiles["file2.txt"] {
		t.Error("file2.txt should NOT be dirty")
	}

	vd.Delete("file2.txt")
	if !vd.DirtyFiles["file2.txt"] {
		t.Error("file2.txt should be dirty (marked for deletion)")
	}

	err = vd.PersistTo(tempDir)
	if err != nil {
		t.Fatalf("PersistTo failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "file1.txt")); os.IsNotExist(err) {
		t.Error("file1.txt not persisted")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "file2.txt")); !os.IsNotExist(err) {
		t.Error("file2.txt should have been deleted")
	}
}

func TestVirtualDisk_NewFeatures(t *testing.T) {
	vd := NewVirtualDisk()

	if vd.FreeSpace() != MaxDiskBytes {
		t.Errorf("Initial FreeSpace = %d, expected %d", vd.FreeSpace(), MaxDiskBytes)
	}

	vd.Write("test.txt", []byte{1, 2, 3})
	if vd.FreeSpace() != MaxDiskBytes-3 {
		t.Errorf("FreeSpace after write = %d, expected %d", vd.FreeSpace(), MaxDiskBytes-3)
	}

	created, modified, err := vd.GetMeta("test.txt")
	if err != nil {
		t.Errorf("GetMeta failed: %v", err)
	}
	if created.IsZero() || modified.IsZero() {
		t.Error("Timestamps zero")
	}

	vd.Write("a.txt", []byte{1})
	vd.Write("c.txt", []byte{1})
	vd.Write("b.txt", []byte{1})

	list := vd.List()
	expected := []string{"a.txt", "b.txt", "c.txt", "test.txt"}
	if !reflect.DeepEqual(list, expected) {
		t.Errorf("List = %v, expected %v", list, expected)
	}

	err = vd.Delete("b.txt")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
	if _, ok := vd.Files["b.txt"]; ok {
		t.Error("File b.txt still exists after delete")
	}

	list = vd.List()
	expected = []string{"a.txt", "c.txt", "test.txt"}
	if !reflect.DeepEqual(list, expected) {
		t.Errorf("List after delete = %v, expected %v", list, expected)
	}

	err = vd.Delete("missing.txt")
	if err != ErrFileNotFound {
		t.Errorf("Delete missing file error = %v, expected ErrFileNotFound", err)
	}
}
