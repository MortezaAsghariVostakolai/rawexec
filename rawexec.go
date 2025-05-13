// Package rawexec executes raw machine code binaries in Go with minimal overhead.
//
// WARNING: This package uses unsafe and mmap to execute raw machine code.
// Invalid or malicious binaries may cause crashes, memory corruption, or undefined behavior.
// Ensure binaries are valid for the target architecture (e.g., x86-64) and thoroughly tested.
//
// # Custom Structs for Arguments and Returns
//
// The FuncSignature takes a single uintptr argument, which can point to a user-defined struct
// containing input arguments, output values, or metadata. Users are responsible for defining
// the struct, passing its pointer as the uintptr, and ensuring the binary correctly interprets
// the struct’s layout. The struct can be modified between calls to update inputs or retrieve outputs.
//
// Example:
//
//	type Args struct {
//	    In  [2]float64 // Input values
//	    Out float64    // Output value
//	}
//	args := &Args{In: [2]float64{1000.0, 2456.0}}
//	caller.Call(uintptr(unsafe.Pointer(args)))
//	// Binary updates args.Out
package rawexec

import (
	"fmt"
	"unsafe"
)

// New allocates executable memory for the given binary code and returns a Caller.
// The binary must be valid machine code for the target architecture (e.g., x86-64).
// The Caller must be freed with Free to avoid memory leaks.
func New(bin []byte) (*Caller, error) {
	gfs := goFuncvalSize()
	if len(bin) > (1<<30)-gfs { // 1GB limit for safety
		return nil, fmt.Errorf("binary too large: %d bytes", len(bin))
	}
	codeBuf, err := alloc(len(bin) + gfs)
	if err != nil {
		return nil, fmt.Errorf("alloc failed: %w", err)
	}
	*(*uintptr)(unsafe.Pointer(&codeBuf[0])) = uintptr(unsafe.Pointer(&codeBuf[gfs]))
	copy(codeBuf[gfs:], bin)
	var f FuncSignature
	fPtr := (*FuncSignature)(unsafe.Pointer(&f))
	*fPtr = *(*FuncSignature)(unsafe.Pointer(&codeBuf))
	return &Caller{
		buf:  codeBuf,
		Call: f,
	}, nil
}

// Caller holds executable memory and a callable function pointer.
type Caller struct {
	buf  []byte        // mmap-allocated buffer
	Call FuncSignature // Callable function
}

// Free releases the allocated memory.
func (c Caller) Free() error {
	if err := free(c.buf); err != nil {
		return fmt.Errorf("free failed: %w", err)
	}
	return nil
}

// FuncSignature is the function signature for raw binaries, taking a single uintptr argument.
// The argument typically points to a user-defined struct containing inputs and outputs.
// The binary must interpret the struct’s layout correctly.
//
// Example:
//
//	type Args struct {
//	    In  [2]float64
//	    Out float64
//	}
//	args := &Args{In: [2]float64{1000.0, 2456.0}}
//	caller.Call(uintptr(unsafe.Pointer(args)))
type FuncSignature func(arg uintptr)

// goFuncvalSize returns the size of Go’s funcval structure.
// This ensures proper alignment and compatibility with the Go runtime.
func goFuncvalSize() int {
	var dummy FuncSignature = func(arg uintptr) {}
	return int(unsafe.Sizeof(dummy))
}
