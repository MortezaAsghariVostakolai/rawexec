//go:build windows
// +build windows

package rawexec

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// alloc allocates readable, writable, and executable memory for the given size.
func alloc(size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid size: %d", size)
	}

	// Allocate memory with execute, read, and write permissions
	addr, err := windows.VirtualAlloc(0, uintptr(size), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_EXECUTE_READWRITE)
	if err != nil {
		return nil, fmt.Errorf("VirtualAlloc failed: %w", err)
	}

	// Create a slice backed by the allocated memory
	// Use a large capacity to avoid bounds issues, but limit length to size
	return (*[1 << 30]byte)(unsafe.Pointer(addr))[:size:size], nil
}

// free releases the allocated memory.
func free(b []byte) error {
	if len(b) == 0 {
		return nil // Nothing to free
	}

	// Release the memory starting at the base address
	err := windows.VirtualFree(uintptr(unsafe.Pointer(&b[0])), 0, windows.MEM_RELEASE)
	if err != nil {
		return fmt.Errorf("VirtualFree failed: %w", err)
	}
	return nil
}