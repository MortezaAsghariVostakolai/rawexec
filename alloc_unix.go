//go:build linux || darwin
// +build linux darwin

package rawexec

import (
	"fmt"
	"syscall"
)

// alloc allocates readable, writable, and executable memory for the given size.
func alloc(size int) ([]byte, error) {
	prot := syscall.PROT_READ | syscall.PROT_WRITE | syscall.PROT_EXEC
	flags := syscall.MAP_PRIVATE | syscall.MAP_ANON
	fd := -1
	offset := 0
	b, err := syscall.Mmap(fd, int64(offset), size, prot, flags)
	if err != nil {
		return nil, fmt.Errorf("mmap failed: %w", err)
	}
	return b, nil
}

// free releases the allocated memory.
func free(b []byte) error {
	if err := syscall.Munmap(b); err != nil {
		return fmt.Errorf("munmap failed: %w", err)
	}
	return nil
}
