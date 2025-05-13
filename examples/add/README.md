# Add Example

This example demonstrates using `rawexec` to execute an `amd64` binary that adds two `float64` values from a custom `Args` struct and stores the result. It assembles `sse_add_amd64.asm` at runtime using the Flat Assembler (FASM).

## Prerequisites
- Install FASM: https://flatassembler.net
- Ensure `sse_add_amd64.asm` is in the current directory.

## Files
- `main.go`: Go program that assembles and runs the binary.
- `sse_add_amd64.asm`: FASM source code for the `amd64` addition binary.

## Running
```bash
go run main.go
```

Expected output:
```
Result: 3456
Updated Result: 5456
```