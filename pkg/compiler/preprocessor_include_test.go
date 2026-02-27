package compiler

import (
	"gocpu/lib"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreprocessIncludes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "preprocess-include-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	userHeaderPath := filepath.Join(tmpDir, "user.h")
	userHeaderContent := "int user_function(void);"
	if err := os.WriteFile(userHeaderPath, []byte(userHeaderContent), 0644); err != nil {
		t.Fatalf("Failed to write user.h: %v", err)
	}

	mainSrcContent := `#include "user.h"
#include <stdio.c>
`

	systemHeaderContentBytes, err := lib.CFiles.ReadFile("_c_files/stdio.c")
	if err != nil {
		t.Fatalf("Failed to read embedded stdio.c: %v", err)
	}
	systemHeaderContent := string(systemHeaderContentBytes)

	processed, err := Preprocess(mainSrcContent, tmpDir)
	if err != nil {
		t.Fatalf("Preprocess failed: %v", err)
	}

	if !strings.Contains(processed, userHeaderContent) {
		t.Errorf("Processed content does not contain user header content")
	}

	if !strings.Contains(processed, systemHeaderContent) {
		t.Errorf("Processed content does not contain system header content")
	}
}
