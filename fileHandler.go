package main

import (
	"bufio"
	"fmt"
	"os"
)

func readDomains() (res []result) {

	file, err := os.Open("domains.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return []result{}
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Read each line
	for scanner.Scan() {
		line := scanner.Text()
		res = append(res, result{domain: line})
	}

	// Check for any scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}
	return
}
