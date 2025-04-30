package numregex

import (
	"bufio" // For efficient file reading
	"fmt"   // For error formatting
	"os"    // For file operations
	"sort"
	"strconv"
	"strings"
)

// TrieNode represents a node in the prefix tree (Trie).
type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool // Marks the end of a complete number sequence
}

// NewTrieNode creates and initializes a new TrieNode.
func NewTrieNode() *TrieNode {
	return &TrieNode{children: make(map[rune]*TrieNode)}
}

// Insert adds a number string into the Trie.
func (n *TrieNode) Insert(s string) {
	node := n
	for _, r := range s {
		child, exists := node.children[r]
		if !exists {
			child = NewTrieNode()
			node.children[r] = child
		}
		node = child
	}
	node.isEnd = true
}

// escapeCharClass escapes characters special inside regex character classes ([]).
func escapeCharClass(char rune) string {
	s := string(char)
	// Escape characters special within []: \, ], -, ^
	if strings.ContainsRune("]-\\^", char) {
		return `\` + s
	}
	return s
}

// escapeFinalRegex escapes regex metacharacters in the final output string.
func escapeFinalRegex(s string) string {
	// Metacharacters to escape (excluding |()?[] which are structural)
	// Also escape backslash itself.
	metaChars := `. + * { } ^ $ \`
	var builder strings.Builder
	for _, r := range s {
		char := string(r)
		shouldEscape := false
		// Check if the rune is one of the metacharacters
		if strings.ContainsRune(metaChars, r) {
			shouldEscape = true
		}
		if shouldEscape {
			builder.WriteString(`\`)
		}
		builder.WriteString(char)
	}
	return builder.String()
}

// buildRegexRecursive traverses the Trie and constructs the regex pattern,
// attempting to mimic regexgen.py's output formatting (char classes, optionals).
func buildRegexRecursive(node *TrieNode) string {
	// --- Get parts from children ---
	if len(node.children) == 0 {
		if node.isEnd {
			return ""
		}
		return "" // Should be unreachable
	}

	// Get child keys (runes) and sort them for deterministic regex output.
	keys := make([]rune, 0, len(node.children))
	for r := range node.children {
		keys = append(keys, r)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	var childParts []string // Regex strings generated for each child branch
	allChildrenAreSingleChars := true
	var singleCharsOnly []string // Store the single chars (as strings)

	for _, r := range keys {
		childNode := node.children[r]
		char := string(r)
		childRegex := buildRegexRecursive(childNode) // Recursive call

		// --- Combine char with childRegex ---
		var combinedPart string
		if childRegex == "" {
			combinedPart = char
		} else {
			needsGroup := strings.Contains(childRegex, "|") ||
				(childNode.isEnd && len(childNode.children) > 0) ||
				(len(childRegex) > 1 && strings.HasSuffix(childRegex, "?"))

			if childNode.isEnd && len(childNode.children) > 0 {
				combinedPart = char + "(?:|" + childRegex + ")"
			} else if needsGroup {
				combinedPart = char + "(?:" + childRegex + ")"
			} else {
				combinedPart = char + childRegex
			}
		}
		childParts = append(childParts, combinedPart)

		if len(combinedPart) != 1 {
			allChildrenAreSingleChars = false
		} else {
			singleCharsOnly = append(singleCharsOnly, combinedPart)
		}
	} // End loop over children

	// --- Format the combination of childParts based on node.isEnd ---
	isOptional := node.isEnd

	if len(childParts) == 0 {
		if isOptional {
			return ""
		}
		return "" // Should be unreachable
	}

	var middle string
	if isOptional {
		// --- Case 1: Optional needed ---
		if len(childParts) == 1 {
			part := childParts[0]
			if len(part) == 1 {
				middle = escapeCharClass(rune(part[0])) + "?"
			} else {
				middle = "(?:" + part + ")?"
			}
		} else if allChildrenAreSingleChars {
			sort.Strings(singleCharsOnly)
			var charClassContent strings.Builder
			for _, c := range singleCharsOnly {
				charClassContent.WriteString(escapeCharClass(rune(c[0])))
			}
			middle = "[" + charClassContent.String() + "]?"
		} else {
			middle = "(?:|" + strings.Join(childParts, "|") + ")"
		}
	} else {
		// --- Case 2: Not optional ---
		if len(childParts) == 1 {
			middle = childParts[0]
		} else if allChildrenAreSingleChars {
			sort.Strings(singleCharsOnly)
			var charClassContent strings.Builder
			for _, c := range singleCharsOnly {
				charClassContent.WriteString(escapeCharClass(rune(c[0])))
			}
			middle = "[" + charClassContent.String() + "]"
		} else {
			middle = "(?:" + strings.Join(childParts, "|") + ")"
		}
	}

	return middle
}

// GenerateRegexFromNumbers reads numbers from the specified file (one per line)
// and creates a simplified Perl-compatible regex string using a Trie approach,
// mimicking regexgen.py output style. Returns the regex string and any error encountered.
func GenerateRegexFromNumbers(filePath string) (string, error) {
	// --- 1. Read numbers from file ---
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening file %s: %w", filePath, err)
	}
	defer file.Close() // Ensure file is closed

	numbers := []int{}
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // Skip empty lines
		}
		num, err := strconv.Atoi(line)
		if err != nil {
			// Optionally, decide whether to skip invalid lines or return an error
			fmt.Printf("Warning: Skipping invalid number '%s' on line %d of %s\n", line, lineNumber, filePath)
			// return "", fmt.Errorf("error converting line %d ('%s') to number in file %s: %w", lineNumber, line, filePath, err)
			continue // Skip this line
		}
		numbers = append(numbers, num)
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	// --- 2. Handle empty input (after reading file) ---
	if len(numbers) == 0 {
		fmt.Printf("Warning: No valid numbers found in file %s. Returning non-matching regex.\n", filePath)
		return "(?!)", nil // Return non-matching regex, but not as an error
	}

	// --- 3. Remove duplicates and prepare strings ---
	uniqueNumbers := make(map[string]struct{})
	hasZero := false
	for _, num := range numbers {
		if num == 0 {
			hasZero = true
			continue
		}
		strNum := strconv.Itoa(num)
		strNum = strings.ReplaceAll(strNum, `\`, `\\`) // Escape backslashes
		uniqueNumbers[strNum] = struct{}{}
	}
	numStrings := make([]string, 0, len(uniqueNumbers))
	for s := range uniqueNumbers {
		numStrings = append(numStrings, s)
	}

	// --- 4. Handle simple cases ---
	if !hasZero && len(numStrings) == 0 {
		if hasZero {
			return "0", nil
		}
		return "(?!)", nil // Should have been caught by len(numbers) == 0 check
	}
	if !hasZero && len(numStrings) == 1 {
		return escapeFinalRegex(numStrings[0]), nil
	}
	if hasZero && len(numStrings) == 0 {
		return "0", nil
	}

	// --- 5. Build the Trie ---
	root := NewTrieNode()
	for _, s := range numStrings {
		root.Insert(s)
	}

	// --- 6. Generate regex parts ---
	var finalParts []string
	if hasZero {
		finalParts = append(finalParts, "0")
	}
	trieRegex := buildRegexRecursive(root)
	if trieRegex != "" {
		finalParts = append(finalParts, trieRegex)
	}

	// --- 7. Format final regex ---
	if len(finalParts) == 0 {
		return "(?!)", nil // Should not happen if numbers were found
	}
	finalRegex := strings.Join(finalParts, "|")

	// --- 8. Escape final string ---
	return escapeFinalRegex(finalRegex), nil
}

