// Package rawexec executes raw machine code binaries in Go with minimal overhead.
//
// It provides two primary types:
//
//   - [ExecutableMemory]: Allocates and manages executable memory via mmap (Unix)
//     or VirtualAlloc (Windows). The memory is automatically released by the GC
//     when the [ExecutableMemory] becomes unreachable. The BaseAddr method
//     returns the starting address, suitable for computing function offsets in
//     shared libraries.
//
//   - [Caller]: Wraps executable memory containing a single function as a typed,
//     callable Go function pointer. The generic parameter Fn specifies the
//     function signature, enabling direct Go-style calls into raw machine code
//     with no allocation overhead.
//
// # Safety
//
// This package uses unsafe pointer manipulation and allocates memory with
// execute permissions. Invalid or malicious binaries may cause crashes, memory
// corruption, or undefined behavior. Ensure binaries are valid for the target
// architecture (x86-64 or 386) and thoroughly tested before use in production.
//
// # Calling Convention
//
// The raw binary is called using Go's standard ABI calling convention. The
// function signature specified by the type parameter Fn determines how
// arguments are passed and results are returned. For example:
//
//   - func(*Args): the binary receives a pointer to Args in RAX/EAX
//   - func(float64) float64: the binary receives a float64 in XMM0 and
//     returns a float64 in XMM0
//   - func(): the binary takes no arguments and returns nothing
//
// Users may define a struct to pass multiple arguments or receive multiple
// return values via a single pointer, but this is a design choice, not a
// requirement of the package. The binary's calling convention must match
// the Go function signature provided to [NewCallable].
//
// Example struct-based calling convention:
//
//	type Args struct {
//	    In  [2]float64
//	    Out float64
//	}
//	caller, _ := rawexec.NewCallable[func(*Args)](binary)
//
// Example register-based calling convention:
//
//	caller, _ := rawexec.NewCallable[func(float64) float64](binary)
//	result := caller.Call(3.14)
//
// # Caller Usage
//
// Use [NewCallable] to load a pre-assembled binary and call it as a typed Go
// function. This is the simplest interface for executing fixed machine code
// routines.
//
//	caller, err := rawexec.NewCallable[func(*Args)](binary)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	args := &Args{In: [2]float64{1.0, 2.0}}
//	caller.Call(args)
//	fmt.Println(args.Out)
//
// # Library Usage
//
// Use [NewLibrary] to load a raw binary containing multiple exported functions
// into executable memory. The returned [Library] provides the base address of
// the loaded code via its embedded [ExecutableMemory]. The caller can compute
// absolute function addresses by adding known offsets (e.g., from a symbol
// table embedded in the binary) to the base address.
//
//	lib, err := rawexec.NewLibrary(mathLibBinary)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	sinAddr := lib.BaseAddr() + sinOffset
//	// Pass sinAddr to JIT-generated code
//
// The memory is automatically freed by the GC when the Library becomes
// unreachable. No explicit cleanup is required.
//
// # ExecutableMemory Usage
//
// Use [NewExecutableMemory] to allocate raw executable memory of a given size.
// This is useful for JIT compilers that generate code at runtime and write it
// directly into the buffer.
//
//	em, err := rawexec.NewExecutableMemory(4096)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Write machine code into em.Buffer() or via em.BaseAddr()
//	// Memory is automatically freed by GC when em is no longer referenced
//
// # Memory Management
//
// Memory allocated by [NewExecutableMemory] and [NewCallable] is automatically
// released by the Go garbage collector when the returned object becomes
// unreachable.
//
// # Platform Support
//
//   - Linux: amd64, 386
//   - macOS: amd64, 386
//   - Windows: amd64, 386
//
// # Limits
//
// The maximum allocation size is [MaxMemSize] (1 GB). Attempting to allocate
// more returns an error.
package rawexec

import (
	"fmt"
	"runtime"
	"unsafe"
)

const (
	// MaxMemSize is the maximum size in bytes for a single executable memory
	// allocation. Attempting to allocate more returns an error.
	MaxMemSize = 1 << 30
)

// NewCallable allocates executable memory for the given binary and returns a
// Caller with a typed function pointer.
//
// The type parameter Fn specifies the Go function signature. The binary must
// be valid machine code for the target architecture. The binary receives a
// single pointer argument (in RAX/EAX) to a user-defined struct.
//
// The returned Caller embeds an [ExecutableMemory] and provides access to
// BaseAddr for computing function offsets. The memory is automatically freed
// by the GC when the Caller becomes unreachable.
func NewCaller[Fn any](bin []byte) (*Caller[Fn], error) {
	if len(bin) < 1 {
		return nil, fmt.Errorf("can use empty binary")
	}

	gfs := goFuncvalSize[Fn]()
	size := len(bin) + gfs
	em, err := NewExecutableMemory(size)
	if err != nil {
		return nil, err
	}

	*(*uintptr)(unsafe.Pointer(&em.buf[0])) = uintptr(unsafe.Pointer(&em.buf[gfs]))
	copy(em.buf[gfs:], bin)
	var f Fn
	fPtr := (*Fn)(unsafe.Pointer(&f))
	*fPtr = *(*Fn)(unsafe.Pointer(&em.buf))

	caller := &Caller[Fn]{
		ExecutableMemory: em,
		Call:             f,
	}

	return caller, nil
}

// Caller wraps executable memory as a typed, callable Go function pointer.
//
// The Call field is a function of type Fn that can be called directly from Go
// using standard Go calling conventions. Arguments and return values are passed
// according to the Go ABI for the specified signature.
//
// Caller embeds [*ExecutableMemory], providing access to BaseAddr for computing
// function offsets within the binary.
type Caller[Fn any] struct {
	*ExecutableMemory
	Call Fn
}

// goFuncvalSize returns the size in bytes of Go's internal function value
// representation for the given function signature. This is used to properly
// allocate and align executable memory so the function pointer is recognized
// by Go's runtime as a valid callable function.
func goFuncvalSize[Fn any]() int {
	var dummy Fn
	return int(unsafe.Sizeof(dummy))
}

// NewLibrary loads a raw binary into executable memory and returns a Library.
//
// The binary may contain multiple functions, data tables, or other resources.
// The returned Library provides the base address of the loaded code via
// BaseAddr. Function addresses are computed by adding known offsets to the
// base address.
//
// The memory is automatically freed by the GC when the Library becomes
// unreachable.
func NewLibrary(bin []byte) (*Library, error) {
	em, err := NewExecutableMemory(len(bin))
	if err != nil {
		return nil, err
	}
	copy(em.buf, bin)
	lib := &Library{ExecutableMemory: em}
	return lib, nil
}

// Library holds executable memory containing a loaded binary with one or more
// exported functions.
//
// Library embeds [*ExecutableMemory], providing access to BaseAddr for
// computing absolute function addresses within the loaded binary. This is
// suitable for shared libraries where the caller knows the offsets of
// individual functions (e.g., via an embedded symbol table).
//
// The memory is automatically freed by the GC when the Library becomes
// unreachable.
type Library struct {
	*ExecutableMemory
}

// NewExecutableMemory allocates a block of executable memory of the given size.
//
// The returned memory is zero-initialized and can be written to via the buffer
// slice or through the base address returned by BaseAddr. The memory is
// automatically freed by the GC when the ExecutableMemory becomes unreachable.
//
// Size must be positive and not exceed [MaxMemSize]. Returns an error if
// allocation fails or the size is out of range.
func NewExecutableMemory(size int) (*ExecutableMemory, error) {
	if size > MaxMemSize {
		return nil, fmt.Errorf("size out of range (1,%d)", MaxMemSize)
	}

	codeBuf, err := alloc(size)
	if err != nil {
		return nil, fmt.Errorf("alloc failed: %w", err)
	}

	em := &ExecutableMemory{
		buf:      codeBuf,
		baseAddr: uintptr(unsafe.Pointer(&codeBuf[0])),
	}

	runtime.AddCleanup(em, func(s []byte) {
		free(s)
	}, em.buf)

	return em, nil
}

// ExecutableMemory represents a block of allocated executable memory.
//
// The memory is automatically released by the garbage collector when the
// ExecutableMemory becomes unreachable. For deterministic cleanup, call Free
// directly.
type ExecutableMemory struct {
	buf      []byte
	baseAddr uintptr
}

// BaseAddr returns the starting address of the executable memory.
func (em *ExecutableMemory) BaseAddr() uintptr {
	return em.baseAddr
}
