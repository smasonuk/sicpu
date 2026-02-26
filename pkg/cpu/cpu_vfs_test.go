package cpu

import (
	"testing"
)

func TestVFS_NewCommands(t *testing.T) {
	c := NewCPU()

	// Helper to write ASCII string to RAM (1 byte per char + null)
	writeString := func(addr uint16, s string) {
		for i := 0; i < len(s); i++ {
			c.Memory[addr+uint16(i)] = s[i]
		}
		c.Memory[addr+uint16(len(s))] = 0
	}

	// Helper to read ASCII string from RAM
	readString := func(addr uint16) string {
		var s []byte
		for i := 0; i < 100; i++ {
			val := c.Memory[addr+uint16(i)]
			if val == 0 {
				break
			}
			s = append(s, val)
		}
		return string(s)
	}

	// 1. Check Free Space (CMD 6)
	c.WriteMem(0xFF10, 6)
	if c.Read16(0xFF14) != 0 {
		t.Errorf("Free Space: Expected Status=0, got %d", c.Read16(0xFF14))
	}
	// Initial free space should be MaxDiskBytes (1474560)
	low := c.Read16(0xFF13)
	high := c.Read16(0xFF15)
	freeSpace := int(uint32(high)<<16 | uint32(low))
	if freeSpace != 1474560 { // MaxDiskBytes
		t.Errorf("Free Space: Expected 1474560, got %d", freeSpace)
	}

	// 2. Write File "test.txt" with 1 byte (0xAA)
	writeString(0x1000, "test.txt")
	c.Memory[0x2000] = 0xAA
	c.Write16(0xFF11, 0x1000) // NamePtr
	c.Write16(0xFF12, 0x2000) // BufPtr
	c.Write16(0xFF13, 1)      // Length = 1 byte
	c.WriteMem(0xFF10, 2)     // Write
	if c.Read16(0xFF14) != 0 {
		t.Fatalf("Write: Failed with status %d", c.Read16(0xFF14))
	}

	// 3. Get Metadata (CMD 7)
	c.Write16(0xFF12, 0x3000) // Buffer for meta
	c.WriteMem(0xFF10, 7)
	if c.Read16(0xFF14) != 0 {
		t.Errorf("GetMeta: Failed with status %d", c.Read16(0xFF14))
	}
	// Check year (stored as LE uint16 at 0x3000)
	year := c.Read16(0x3000)
	if year < 2023 {
		t.Errorf("GetMeta: Year %d seems invalid", year)
	}

	// 4. List (CMD 5)
	// Create another file
	writeString(0x1000, "a.txt")
	c.Memory[0x2000] = 0xBB
	c.Write16(0xFF11, 0x1000)
	c.Write16(0xFF12, 0x2000)
	c.Write16(0xFF13, 1)
	c.WriteMem(0xFF10, 2) // Write

	files := []string{}
	c.Write16(0xFF12, 0x4000) // Buffer for name

	c.WriteMem(0xFF10, 5)
	if c.Read16(0xFF14) != 0 {
		t.Errorf("List 1: Failed with status %d", c.Read16(0xFF14))
	}
	files = append(files, readString(0x4000))

	c.WriteMem(0xFF10, 5)
	if c.Read16(0xFF14) != 0 {
		t.Errorf("List 2: Failed with status %d", c.Read16(0xFF14))
	}
	files = append(files, readString(0x4000))

	c.WriteMem(0xFF10, 5)
	if c.Read16(0xFF14) != 5 {
		t.Errorf("List 3: Expected Status=5 (DirEnd), got %d", c.Read16(0xFF14))
	}

	if len(files) != 2 {
		t.Fatalf("List: Expected 2 files, got %d", len(files))
	}
	if files[0] != "a.txt" {
		t.Errorf("List: Expected first file 'a.txt', got '%s'", files[0])
	}
	if files[1] != "test.txt" {
		t.Errorf("List: Expected second file 'test.txt', got '%s'", files[1])
	}

	// 5. Delete (CMD 4)
	writeString(0x1000, "test.txt")
	c.Write16(0xFF11, 0x1000)
	c.WriteMem(0xFF10, 4)
	if c.Read16(0xFF14) != 0 {
		t.Errorf("Delete: Failed with status %d", c.Read16(0xFF14))
	}

	// Verify deletion by trying to read it
	c.WriteMem(0xFF10, 1)
	if c.Read16(0xFF14) != 1 { // Not Found
		t.Errorf("Verify Delete: Expected Status=1 (NotFound), got %d", c.Read16(0xFF14))
	}
}
