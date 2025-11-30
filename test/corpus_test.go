package test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mikeschinkel/go-cliutil"
)

// TestFuzzCorpus reads each fuzz corpus file and tests it with timeout detection
func TestFuzzCorpus(t *testing.T) {
	// Test ParseVerbosity corpus
	testCorpus(t, "FuzzParseVerbosity", func(input string) error {
		verbosity, err := strconv.Atoi(input)
		if err != nil {
			return fmt.Errorf("invalid input: %w", err)
		}
		_, _ = cliutil.ParseVerbosity(verbosity)
		return nil
	})

	// Test NewCLIOptions corpus
	testCorpus(t, "FuzzNewCLIOptions", func(input string) error {
		parts := strings.Split(input, "\n")
		if len(parts) < 5 {
			return fmt.Errorf("invalid input format")
		}

		quiet := parts[0] == "true"
		verbosity, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid verbosity: %w", err)
		}
		timeout, err := strconv.Atoi(parts[2])
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
		dryRun := parts[3] == "true"
		force := parts[4] == "true"

		_, _ = cliutil.NewCLIOptions(cliutil.CLIOptionsArgs{
			Quiet:     &quiet,
			Verbosity: &verbosity,
			Timeout:   &timeout,
			DryRun:    &dryRun,
			Force:     &force,
		})
		return nil
	})
}

func testCorpus(t *testing.T, fuzzFuncName string, testFunc func(string) error) {
	corpusDir := filepath.Join("testdata", "fuzz", fuzzFuncName)
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		// No corpus directory is fine - fuzzing hasn't been run yet
		return
	}

	infiniteLoops := []string{}
	parseErrors := []string{}
	successes := []string{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Read the fuzz corpus file
		path := filepath.Join(corpusDir, entry.Name())
		f, err := os.Open(path)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		lineNum := 0
		var input string
		for scanner.Scan() {
			lineNum++
			if lineNum == 2 { // Second line contains the input
				line := scanner.Text()
				// Parse the Go format from corpus file
				if strings.HasPrefix(line, "int(") && strings.HasSuffix(line, ")") {
					// For int inputs like "int(5)"
					input = line[4 : len(line)-1]
				} else if strings.HasPrefix(line, "string(") && strings.HasSuffix(line, ")") {
					// For string inputs
					strLiteral := line[7 : len(line)-1]
					unquoted, err := strconv.Unquote(strLiteral)
					if err != nil {
						break
					}
					input = unquoted
				}
				break
			}
		}
		_ = f.Close()

		if input == "" {
			continue
		}

		// Test this input with timeout
		done := make(chan struct{})
		var testErr error

		go func() {
			defer func() {
				if r := recover(); r != nil {
					testErr = fmt.Errorf("PANIC: %v", r)
				}
				close(done)
			}()

			testErr = testFunc(input)
		}()

		select {
		case <-done:
			// Test completed
			if testErr != nil {
				parseErrors = append(parseErrors, entry.Name())
			} else {
				successes = append(successes, entry.Name())
			}
		case <-time.After(10 * time.Second):
			infiniteLoops = append(infiniteLoops, entry.Name())
			t.Errorf("%-20s INFINITE LOOP: %q", entry.Name(), input)
		}
	}

	if len(infiniteLoops) > 0 {
		t.Fatalf("Found %d infinite loop(s) in %s", len(infiniteLoops), fuzzFuncName)
	}
}
