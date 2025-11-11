package cliutil

// Exit codes for CLI applications following lifecycle progression.
// Lower numbers indicate earlier failures in the application startup sequence.
//
// These codes are designed to help both users and scripts understand where
// in the application lifecycle a failure occurred:
//   - 1: Failed parsing command-line arguments
//   - 2: Failed loading configuration file(s)
//   - 3: Failed validating configuration content
//   - 4: Expected/handled error during execution
//   - 5: Unexpected/unhandled error during execution
//   - 6: Failed to initialize logging infrastructure
//
// Scripts can use these codes to determine appropriate retry/recovery strategies:
//   - Exit 1-3: Likely user/config error, fix and retry immediately
//   - Exit 4: Known error condition, check logs, may be retryable
//   - Exit 5: Unexpected error, investigate before retry
//   - Exit 6: Infrastructure failure, check system resources
//
// Note: Exit codes 128 and above are reserved for signal-related exits.
// See: https://tldp.org/LDP/abs/html/exitcodes.html

//goland:noinspection GoUnusedConst
const (
	ExitSuccess             = 0 // Successful execution
	ExitOptionsParseError   = 1 // Command-line option parsing failed
	ExitConfigLoadError     = 2 // Configuration file loading failed
	ExitConfigParseError    = 3 // Configuration parsing/validation failed
	ExitKnownRuntimeError   = 4 // Expected/known runtime error during execution
	ExitUnknownRuntimeError = 5 // Unexpected/unknown runtime error
	ExitLoggerSetupError    = 6 // Logger initialization failed
)
