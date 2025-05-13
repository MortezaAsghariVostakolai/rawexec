// Package rawexec_test contains tests for the rawexec package across supported architectures.
package rawexec_test

import (
	"fmt"
	"runtime"
	"testing"
	"unsafe"

	"github.com/MortezaAsghariVostakolai/rawexec"
)

// Args holds input and output values for testing.
type Args struct {
	In  [2]float64 // Input values
	Out float64    // Output value
}

var (
	// addSSEi386Binary is i386 machine code that adds two float64s from Args.In
	// and stores the result in Args.Out. It uses SSE instructions and expects the
	// Args struct pointer at [esp+4], per Go's 386 ABI.
	addSSEi386Binary = []byte{
		0x8B, 0x44, 0x24, 0x04, //       mov eax, [esp+4]       ; Load &In[0] into eax
		0xF2, 0x0F, 0x10, 0x00, //       movsd xmm0, [eax]      ; Load In[0] into XMM0
		0xF2, 0x0F, 0x58, 0x40, 0x08, // addsd xmm0, [eax+8]    ; Add In[1] to XMM0
		0xF2, 0x0F, 0x11, 0x40, 0x10, // movsd [eax+16], xmm0   ; Store result in Out
		0xC3, //                         ret                    ; Return
	}

	// addSSEAmd64Binary is x86-64 machine code that adds two float64s from Args.In
	// and stores the result in Args.Out. It uses SSE instructions and expects RAX
	// to point to an Args struct.
	addSSEAmd64Binary = []byte{
		0xF2, 0x0F, 0x10, 0x00, //       movsd xmm0, [rax]      ; Load In[0] into XMM0
		0xF2, 0x0F, 0x58, 0x40, 0x08, // addsd xmm0, [rax+8]    ; Add In[1] to XMM0
		0xF2, 0x0F, 0x11, 0x40, 0x10, // movsd [rax+16], xmm0   ; Store result in Out
		0xC3, //                         ret                    ; Return
	}
)

// getTestBinary returns the appropriate binary for the current architecture.
func getTestBinary() ([]byte, error) {
	switch runtime.GOARCH {
	case "386":
		return addSSEi386Binary, nil
	case "amd64":
		return addSSEAmd64Binary, nil
	default:
		return nil, fmt.Errorf("test for `%s` architecture is not provided yet", runtime.GOARCH)
	}
}

func TestCaller(t *testing.T) {
	// Select binary for current architecture
	bin, err := getTestBinary()
	if err != nil {
		t.Skip(err)
	}

	// Create Caller with binary
	caller, err := rawexec.New(bin)
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}
	defer caller.Free()

	// Test first call
	args := &Args{In: [2]float64{1000.0, 2456.0}}
	caller.Call(uintptr(unsafe.Pointer(args)))

	if args.Out != 3456.0 {
		t.Errorf("expected 3456.0, got %v", args.Out)
	}

	// Test updated inputs
	args.In[0] = 2000.0
	args.In[1] = 3456.0
	caller.Call(uintptr(unsafe.Pointer(args)))
	if args.Out != 5456.0 {
		t.Errorf("expected 5456.0, got %v", args.Out)
	}
}

func TestInvalidBinarySize(t *testing.T) {
	// Test oversized binary
	bin := make([]byte, 1<<30) // Too large
	_, err := rawexec.New(bin)
	if err == nil {
		t.Fatal("expected error for oversized binary")
	}
}

func BenchmarkCaller(b *testing.B) {
	// Select binary for current architecture
	bin, err := getTestBinary()
	if err != nil {
		b.Skip(err)
	}

	// Create Caller with binary
	caller, err := rawexec.New(bin)
	if err != nil {
		b.Fatal(err)
	}
	defer caller.Free()

	// Benchmark calling the binary
	args := &Args{In: [2]float64{1000.0, 2456.0}}
	for i := 0; i < b.N; i++ {
		caller.Call(uintptr(unsafe.Pointer(args)))
	}
}
