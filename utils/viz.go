package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

// Entry represents a single log entry for visualization.
type Entry struct {
	Timestamp string
	ID        string
	Message   string
}

// TestLog represents the log of a single test, including all entries and their metadata.
type TestLog struct {
	Name    string
	Status  string
	Entries []Entry
}

// ReadEntries parses log entries from an io.Reader and organizes them into TestLog structures.
func ReadEntries(reader io.Reader) []TestLog {
	var testLogs []TestLog
	var currentLog *TestLog

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Start of a new test run
		if strings.HasPrefix(line, "=== RUN") {
			if currentLog != nil {
				testLogs = append(testLogs, *currentLog)
			}
			currentLog = &TestLog{
				Name:    strings.TrimSpace(line[len("=== RUN "):]),
				Entries: []Entry{},
			}
			continue
		}

		// Status of the test
		if strings.HasPrefix(line, "--- PASS:") || strings.HasPrefix(line, "--- FAIL:") {
			if currentLog != nil {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					currentLog.Status = fields[1]
				}
			}
			continue
		}

		// Log entry pattern
		entryPattern := regexp.MustCompile(`([0-9:.]+) \[(\d+)\] (.*)`)
		matches := entryPattern.FindStringSubmatch(line)
		if len(matches) == 4 {
			if currentLog != nil {
				currentLog.Entries = append(currentLog.Entries, Entry{
					Timestamp: matches[1],
					ID:        matches[2],
					Message:   matches[3],
				})
			}
		}
	}

	if currentLog != nil {
		testLogs = append(testLogs, *currentLog)
	}

	return testLogs
}

// PrintLogs displays all logs in a formatted table using the go-pretty/table package.
func PrintLogs(logs []TestLog) {
	for _, log := range logs {
		fmt.Printf("Test Name: %s - Status: %s\n", log.Name, log.Status)
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Timestamp", "ID", "Message"})
		for _, entry := range log.Entries {
			t.AppendRow([]interface{}{entry.Timestamp, entry.ID, entry.Message})
		}
		t.Render()
		fmt.Println("-------")
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: viz <logfile>")
		os.Exit(1)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalf("Error opening file: %s", err)
	}
	defer file.Close()

	testLogs := ReadEntries(file)
	PrintLogs(testLogs)
}
