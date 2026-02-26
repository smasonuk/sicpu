package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Preprocess scans the source code for `#include "filename"` and `#define NAME VALUE` directives.
// It replaces includes with file content and substitutes defines.
// It handles nested includes and prevents circular dependencies (include loops).
func Preprocess(src string, baseDir string) (string, error) {
	// We use a map to track the current include stack to prevent cycles.
	// We use a map to track definitions.
	defines := make(map[string]string)
	return preprocessRecursive(src, baseDir, make(map[string]bool), make(map[string]bool), defines)
}

func preprocessRecursive(src string, baseDir string, visitedStack map[string]bool, alreadyProcessed map[string]bool, defines map[string]string) (string, error) {
	lines := strings.Split(src, "\n")
	var result strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle #define
		if strings.HasPrefix(trimmed, "#define") {
			// Expected format: #define NAME VALUE
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				name := parts[1]
				value := strings.Join(parts[2:], " ")

				// Perform substitution on the value itself using existing defines
				// This handles nested definitions like:
				// #define A 10
				// #define B (A + 5) -> (10 + 5)
				value = applyDefines(value, defines)

				defines[name] = value
			}
			// Replace with empty line to preserve line count roughly
			result.WriteString("\n")
			continue
		}

		if strings.HasPrefix(trimmed, "#include") {
			// Expected format: #include "filename"
			parts := strings.SplitN(trimmed, "\"", 3)
			if len(parts) < 3 {
				// Maybe using <filename>? Not supported per requirement.
				return "", fmt.Errorf("invalid include directive: %s", line)
			}
			filename := parts[1]

			// Resolve path
			// Priority 1: Relative to the current file's directory (standard C behavior)
			fullPath := filepath.Join(baseDir, filename)

			// Check if file exists at relative path
			_, err := os.Stat(fullPath)
			if os.IsNotExist(err) {
				// Priority 2: Relative to CWD / Project Root
				// This allows including "lib/vfs.c" from anywhere if running from root
				cwdPath, absErr := filepath.Abs(filename)
				if absErr == nil {
					if _, err := os.Stat(cwdPath); err == nil {
						fullPath = cwdPath
					}
				}
			}

			absPath, err := filepath.Abs(fullPath)
			if err != nil {
				return "", err
			}

			// Check for cycles in the current stack
			if visitedStack[absPath] {
				return "", fmt.Errorf("circular include detected: %s", filename)
			}

			// Check if this file has already been processed in a different branch of the include tree
			// If so, we can skip reprocessing it to save time, since the result should be the same.
			if alreadyProcessed[absPath] {
				continue
			}
			alreadyProcessed[absPath] = true

			// Read file
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return "", fmt.Errorf("failed to read included file %s (path: %s): %v", filename, fullPath, err)
			}

			// Create a new stack copy for the recursive call to allow diamond dependencies
			newStack := make(map[string]bool)
			for k, v := range visitedStack {
				newStack[k] = v
			}
			newStack[absPath] = true

			// Recursively process the included file
			// The baseDir for the included file is its own directory
			// We pass the same 'defines' map to accumulate definitions across files.
			includeDir := filepath.Dir(fullPath)
			processedContent, err := preprocessRecursive(string(content),
				includeDir,
				newStack,
				alreadyProcessed,
				defines)
			if err != nil {
				return "", err
			}

			result.WriteString(processedContent)
			result.WriteString("\n")
			continue
		}

		// Regular line: apply substitutions
		processedLine := applyDefines(line, defines)
		result.WriteString(processedLine)
		result.WriteString("\n")
	}
	return result.String(), nil
}

// applyDefines replaces occurrences of keys in defines map with their values in the input string.
// It ensures that replacements only happen on word boundaries and not inside string/char literals.
func applyDefines(input string, defines map[string]string) string {
	if len(defines) == 0 {
		return input
	}

	var sb strings.Builder
	n := len(input)
	i := 0

	for i < n {
		if input[i] == '"' {
			// Start string literal
			sb.WriteByte(input[i])
			i++
			for i < n {
				char := input[i]
				sb.WriteByte(char)
				i++
				if char == '\\' {
					// Escape sequence, consume next char if available
					if i < n {
						sb.WriteByte(input[i])
						i++
					}
				} else if char == '"' {
					break
				}
			}
		} else if input[i] == '\'' {
			// Start char literal
			sb.WriteByte(input[i])
			i++
			for i < n {
				char := input[i]
				sb.WriteByte(char)
				i++
				if char == '\\' {
					if i < n {
						sb.WriteByte(input[i])
						i++
					}
				} else if char == '\'' {
					break
				}
			}
		} else {
			r := rune(input[i])
			if isIdentStart(r) {
				start := i
				for i < n && isIdentPart(rune(input[i])) {
					i++
				}
				word := input[start:i]
				if val, ok := defines[word]; ok {
					sb.WriteString(val)
				} else {
					sb.WriteString(word)
				}
			} else {
				sb.WriteByte(input[i])
				i++
			}
		}
	}
	return sb.String()
}

func isIdentStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

func isIdentPart(r rune) bool {
	return isIdentStart(r) || (r >= '0' && r <= '9')
}
