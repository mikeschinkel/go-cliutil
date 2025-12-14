# Exit Code Conventions for CLI Applications

**Date:** 2025-11-07
**Status:** Accepted
**Authors:** Mike Schinkel

## Context

CLI applications need consistent, meaningful exit codes to help:
- Users understand what went wrong when a command fails
- Scripts make intelligent decisions about retries and error handling
- Debugging by indicating where in the application lifecycle a failure occurred

Prior to this ADR, the XMLUI project's three CLI applications (xmlui, xmlui-localsvr, xmlui-mcp) used inconsistent exit codes:
- Exit code 1 meant different things in each app (runtime error vs options parsing vs config loading)
- Comments in source code didn't match actual implementation
- Only one app (xmluisvr) defined named constants; others used magic numbers
- No clear pattern for what each exit code represented

This inconsistency made it difficult for users to write robust shell scripts that handle errors appropriately.

## Decision

We will use **sequential exit codes following lifecycle progression**, where lower numbers indicate earlier failures in the application startup sequence.

### Standard Exit Codes

```go
const (
    ExitSuccess             = 0 // Successful execution
    ExitOptionsParseError   = 1 // Command-line option parsing failed
    ExitConfigLoadError     = 2 // Configuration file loading failed
    ExitConfigParseError    = 3 // Configuration parsing/validation failed
    ExitKnownRuntimeError   = 4 // Expected/known runtime error during execution
    ExitUnknownRuntimeError = 5 // Unexpected/unknown runtime error
    ExitLoggerSetupError    = 6 // Logger initialization failed
)
```

### Design Principles

1. **Lifecycle Progression**: Exit codes increase as you progress through application initialization
   - 1 = earliest possible failure (can't even parse arguments)
   - 6 = infrastructure setup failure
   - 4-5 = runtime failures (main execution logic)

2. **Sequential Numbering**: Use simple sequential integers (1, 2, 3...) rather than gaps or ranges
   - Easy to remember
   - Simple to document
   - Can add new codes at the end if needed

3. **Known vs Unknown Runtime Errors**: Distinguish between:
   - Exit 4: Expected errors that the application handles gracefully (e.g., database connection refused, file not found)
   - Exit 5: Unexpected errors that indicate bugs or unhandled conditions

4. **Reserved Ranges**: Do not use exit codes 128 and above, which are reserved for signal-related exits in Unix/Linux

### Script-Friendly Error Handling

This progression enables scripts to make informed decisions:

```bash
#!/bin/bash
xmlui some-command
case $? in
    0)
        echo "Success"
        ;;
    1|2|3)
        echo "Configuration or usage error - fix and retry immediately"
        exit 1
        ;;
    4)
        echo "Known error - check logs, may retry with backoff"
        sleep 5
        # Retry logic here
        ;;
    5)
        echo "Unexpected error - requires investigation before retry"
        exit 1
        ;;
    6)
        echo "Infrastructure failure - check system resources"
        exit 1
        ;;
esac
```

## Consequences

### Positive

- **Consistency**: All XMLUI CLI tools use the same exit codes for the same types of failures
- **Debuggability**: Exit code immediately tells you where to look (args? config? runtime?)
- **Script-friendly**: Scripts can distinguish between "retry immediately", "retry with backoff", and "don't retry"
- **Self-documenting**: Named constants make code more readable than magic numbers
- **Future-proof**: Can add new codes at the end (7, 8, etc.) without disrupting existing codes

### Negative

- **Migration required**: All three apps need updates to align with new codes
- **Documentation burden**: Must document exit codes in each app's README/help text
- **Slightly non-traditional**: Exit 1 typically means "generic error" in Unix conventions, but we use it specifically for options parsing

### Neutral

- Apps can define additional exit codes (7+) for app-specific scenarios if needed
- The distinction between exit 4 and 5 requires developers to consciously decide if an error is "known" or "unknown"

## Alternatives Considered

### Alternative 1: Severity Progression (1=most severe)
- Exit 1: Runtime errors (most common/important)
- Exit 2: Config errors
- Exit 3: Options errors

**Rejected because**: Counterintuitive for debugging. If something fails with exit 3, it sounds worse than exit 1, but it's actually an earlier, simpler failure.

### Alternative 2: Category Ranges
- 1-10: Configuration errors
- 11-20: Runtime errors
- 21-30: Network errors

**Rejected because**: Over-engineered for our needs. We only have 6 distinct categories, and gaps make codes harder to remember.

### Alternative 3: HTTP-style Codes
- Exit 400: Bad request (options)
- Exit 404: Config not found
- Exit 500: Server error

**Rejected because**: Clever but confusing. Exit codes and HTTP status codes serve different purposes. Also limits us to specific numbers.

### Alternative 4: Per-app Exit Codes
Keep exit codes in each app's common package, no shared constants.

**Rejected because**: Defeats the purpose of standardization. Scripts that use multiple XMLUI tools would need to handle different exit codes for the same types of failures.

## References

- [Advanced Bash-Scripting Guide: Exit Codes](https://tldp.org/LDP/abs/html/exitcodes.html)
- [GNU Coding Standards: Exit Status](https://www.gnu.org/prep/standards/html_node/Exit-Status.html)
- Prior art: Go tools like `go build`, `git`, `docker` use sequential exit codes

## Notes

This ADR was created as part of standardizing the XMLUI CLI, localsvr, and MCP server applications.

Date: 2025-01-09
