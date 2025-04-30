package main

import (
	"fmt"
	"log" // For fatal errors

	"github.com/numregex" // Replace with your actual module path
)

func main() {
	fmt.Println("--- Regex Generation Example (Reads from Existing temp.txt) ---")

	inputFileName := "temp.txt" // The file to read from

	// --- Generate Regex ---
	fmt.Printf("Reading numbers from: %s\n", inputFileName)
	generatedRegex, err := numregex.GenerateRegexFromNumbers(inputFileName)
	if err != nil {
		// Log a fatal error if the file can't be read or processed
		log.Fatalf("Error generating regex from file '%s': %v", inputFileName, err)
	}

	fmt.Printf("Generated Regex: %s\n", generatedRegex)

	// --- Optional: Test the generated regex ---
	// You can add test cases here if needed, similar to the previous example,
	// assuming you know roughly what numbers are in temp.txt

}

