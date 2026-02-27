package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Macro represents a defined macro, either simple or function-like.
type Macro struct {
	Args []string // Empty for simple macros
	Body string
}

// Preprocess scans the source code for `#include "filename"` and `#define NAME VALUE` directives.
// It replaces includes with file content and substitutes defines.
// It handles nested includes and prevents circular dependencies (include loops).
func Preprocess(src string, baseDir string) (string, error) {
	// We use a map to track the current include stack to prevent cycles.
	// We use a map to track definitions.
	defines := make(map[string]Macro)
	return preprocessRecursive(src, baseDir, make(map[string]bool), make(map[string]bool), defines)
}

func preprocessRecursive(src string, baseDir string, visitedStack map[string]bool, alreadyProcessed map[string]bool, defines map[string]Macro) (string, error) {
	lines := strings.Split(src, "\n")
	var result strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle #define
		if strings.HasPrefix(trimmed, "#define") {
			// Expected format: #define NAME VALUE or #define NAME(ARGS) VALUE
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "#define"))
			if rest == "" {
				continue
			}

			// Parse name. Name ends at space or (.
			nameEnd := 0
			for nameEnd < len(rest) {
				r := rest[nameEnd]
				if r == ' ' || r == '\t' || r == '(' {
					break
				}
				nameEnd++
			}
			name := rest[:nameEnd]
			rest = rest[nameEnd:]

			var args []string
			// Check for function-like macro: #define FOO(a,b) ...
			// Must have '(' immediately after name (no spaces).
			if len(rest) > 0 && rest[0] == '(' {
				// Parse args
				closeParen := strings.Index(rest, ")")
				if closeParen == -1 {
					return "", fmt.Errorf("unterminated macro parameter list")
				}
				argStr := rest[1:closeParen]
				if strings.TrimSpace(argStr) != "" {
					argParts := strings.Split(argStr, ",")
					for _, arg := range argParts {
						args = append(args, strings.TrimSpace(arg))
					}
				}
				rest = rest[closeParen+1:]
			}

			value := strings.TrimSpace(rest)

			// Perform substitution on the value itself using existing defines (for simple macros)
			// For function-like macros, we do this at expansion time.
			// Actually, C preprocessor expands body?
			// Let's keep it simple: Expand simple macros in body immediately.
			// But for function-like, arguments shadow global defines.
			// So applyDefines needs to be careful.
			// For now, we won't pre-expand body of function-like macros to avoid complexity with args.
			if len(args) == 0 {
				value = applyDefines(value, defines)
			}

			defines[name] = Macro{Args: args, Body: value}

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
func applyDefines(input string, defines map[string]Macro) string {
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
				if macro, ok := defines[word]; ok {
					// Found a macro. Check if it's function-like.
					if len(macro.Args) > 0 {
						// Function-like macro expansion
						// Look ahead for '('
						// Skip whitespace
						j := i
						for j < n && (input[j] == ' ' || input[j] == '\t') {
							j++
						}
						if j < n && input[j] == '(' {
							// Parse arguments
							j++ // consume '('
							var args []string
							var currentArg strings.Builder
							parenDepth := 1
							for j < n && parenDepth > 0 {
								if input[j] == '(' {
									parenDepth++
									currentArg.WriteByte(input[j])
								} else if input[j] == ')' {
									parenDepth--
									if parenDepth > 0 {
										currentArg.WriteByte(input[j])
									}
								} else if input[j] == ',' && parenDepth == 1 {
									args = append(args, strings.TrimSpace(currentArg.String()))
									currentArg.Reset()
								} else {
									currentArg.WriteByte(input[j])
								}
								j++
							}
							if parenDepth == 0 {
								args = append(args, strings.TrimSpace(currentArg.String()))

								// Expand body
								if len(args) == len(macro.Args) {
									// Build a single substitution map for all arguments and apply it in one
									// pass. A single pass prevents earlier substitutions from being
									// accidentally re-substituted by later argument names (argument bleeding).
									body := macro.Body
									argMap := make(map[string]Macro, len(macro.Args))
									for k, argName := range macro.Args {
										argMap[argName] = Macro{Body: args[k]}
									}
									body = applyDefines(body, argMap)

									// Then recursively apply globals to the result.
									// Note: This naive recursion might loop if a macro expands to itself
									// (e.g. #define A A). Standard C prevents this by marking the macro as
									// 'disabled' during expansion; we omit that check for simplicity.
									expanded := applyDefines(body, defines)
									sb.WriteString(expanded)

									i = j
									continue
								}
							}
						}
						// If not followed by '(', treat as normal identifier (don't expand)
						sb.WriteString(word)
					} else {
						// Simple macro
						sb.WriteString(macro.Body)
					}
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
