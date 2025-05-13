//go:build amd64
// +build amd64

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
	caller, err := rawexec.New(bin)
	if err != nil {
		log.Fatal(err)
	}
	defer caller.Free()

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
