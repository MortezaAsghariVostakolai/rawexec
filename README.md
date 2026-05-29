# rawexec

rawexec executes raw machine code binaries in Go with minimal overhead, ideal for performance-critical workloads like transpiled binaries or JIT compilation.

## Features
- Execute raw machine code with near-native performance (x86-64, 386).
- Pass arguments and returns via a user-defined struct pointed to by a single `uintptr`.
- Zero allocations per call, leveraging Go’s concurrency and garbage collection.
- Supports Linux, macOS, and Windows (amd64, 386).

## Why rawexec?

`rawexec` was created to address the performance challenges of genetic programming, where interpreting genomes as CPU instructions is computationally intensive. Traditional approaches, like transpiling to C/Go or using Cgo, introduced significant overhead, especially for small genomes and frequent calls. `rawexec` enables direct execution of raw binary code in Go, allowing native binaries to be generated and run at runtime with minimal overhead, achieving near-native performance. It's designed for developers optimizing genetic algorithms, JIT compilers, or other performance-critical workloads.

## Installation
```bash
go get github.com/MortezaAsghariVostakolai/rawexec
```

## Dependencies
- Linux/macOS: Uses `syscall` package (standard library).
- Windows: Requires `golang.org/x/sys/windows` for memory allocation.
- Example: Requires Flat Assembler (FASM) for assembling `amd64` binaries.

## Example
The `examples/add` directory demonstrates assembling and executing an `amd64` binary that adds two `float64` values. It requires FASM (https://flatassembler.net).

```go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"unsafe"

	"github.com/MortezaAsghariVostakolai/rawexec"
)

const (
	srcFname = "sse_add_amd64.asm" // Source assembly file
	binFname = "sse_add_amd64.bin" // Output binary file
)

type FnSignature func(uintptr)

// Args holds input and output values for the binary.
type Args struct {
	In  [2]float64 // Input values
	Out float64    // Output value
}

func main() {
	// Use temp directory for output binary
	tempDir := os.TempDir()
	binPath := filepath.Join(tempDir, binFname)

	// Assemble sse_add_amd64.asm to temp binary using FASM
	// Requires FASM to be installed (https://flatassembler.net)
	cmd := exec.Command("fasm", srcFname, binPath)
	if err := cmd.Run(); err != nil {
		log.Fatalf("failed to assemble with fasm: %v", err)
	}

	// Read the generated binary
	bin, err := os.ReadFile(binPath)
	if err != nil {
		log.Fatalf("failed to read binary file: %v", err)
	}
	defer func() {
		// Clean up temporary binary file
		if err := os.Remove(binPath); err != nil {
			log.Printf("failed to delete `%s`: %v", binPath, err)
		}
	}()

	// Create Caller with binary
	caller, err := rawexec.NewCaller[FnSignature](bin)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize Args struct and call binary
	args := &Args{In: [2]float64{1000.0, 2456.0}}
	caller.Call(uintptr(unsafe.Pointer(args)))
	fmt.Printf("Result: %v\n", args.Out) // Expected: 3456

	// Update inputs and call again
	args.In[0] = 2000.0
	args.In[1] = 3456.0
	caller.Call(uintptr(unsafe.Pointer(args)))
	fmt.Printf("Updated Result: %v\n", args.Out) // Expected: 5456
}

```

## Running Tests
Tests in `rawexec_test.go` support `amd64` and `386` using hardcoded binaries, requiring no external dependencies.

```bash
go test -v ./...
```

## Benchmarks
Benchmarks for adding two float64 values using SSE instructions and storing the result were run on:
- **System**: Linux, amd64 and 386
- **CPU**: Intel(R) Core(TM) i7-9750H @ 2.60GHz

Results:
- amd64 (BenchmarkCaller-12): 824763121 iterations, 1.352 ns/op, 0 B/op, 0 allocs/op
- 386 (BenchmarkCaller-12): 819734725 iterations, 1.360 ns/op, 0 B/op, 0 allocs/op

*Note*: Performance depends on the system and workload. Run `go test -bench=.` to measure on your hardware.

## Warning
rawexec uses `unsafe` and `mmap` (Linux/macOS) or `VirtualAlloc` (Windows). Invalid binaries may cause crashes or undefined behavior. Ensure binaries match the target architecture (x86-64 or 386) and struct layout.

## Compatibility
- Go 1.20+
- Linux, macOS, and Windows (amd64, 386)

## License
MIT License. See [LICENSE](LICENSE).