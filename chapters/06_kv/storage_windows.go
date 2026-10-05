//go:build windows

package chapter06

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	genericRead             = 0x80000000
	genericWrite            = 0x40000000
	createAlways            = 2
	openExisting            = 3
	fileBegin               = 0
	fileEnd                 = 2
	fileAttributeNormal     = 0x80
	moveFileReplaceExisting = 0x1
	moveFileWriteThrough    = 0x8
)

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	moveFileExProc = kernel32.NewProc("MoveFileExW")
	deleteFileProc = kernel32.NewProc("DeleteFileW")
)

func readSnapshot(path string) ([]byte, error) {
	handle, err := openFile(path, genericRead, openExisting)
	if err != nil {
		if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) || errors.Is(err, syscall.ERROR_PATH_NOT_FOUND) {
			return nil, errSnapshotNotFound
		}
		return nil, err
	}

	var high int32
	low, err := syscall.SetFilePointer(handle, 0, &high, fileEnd)
	if err != nil {
		return nil, closeWithError(handle, err)
	}
	size := uint64(uint32(high))<<32 | uint64(low)
	maxInt := uint64(^uint(0) >> 1)
	if size > maxInt {
		return nil, closeWithError(handle, fmt.Errorf("snapshot is too large to load: %d bytes", size))
	}
	if _, err := syscall.SetFilePointer(handle, 0, nil, fileBegin); err != nil {
		return nil, closeWithError(handle, err)
	}

	data := make([]byte, int(size))
	for offset := 0; offset < len(data); {
		end := offset + 32*1024
		if end > len(data) {
			end = len(data)
		}
		var read uint32
		if err := syscall.ReadFile(handle, data[offset:end], &read, nil); err != nil {
			return nil, closeWithError(handle, err)
		}
		if read == 0 {
			return nil, closeWithError(handle, io.ErrUnexpectedEOF)
		}
		offset += int(read)
	}
	if err := syscall.CloseHandle(handle); err != nil {
		return nil, fmt.Errorf("close snapshot after reading: %w", err)
	}
	return data, nil
}

func writeSnapshot(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create snapshot directory: %w", err)
	}
	tempPath := path + ".tmp"
	handle, err := openFile(tempPath, genericWrite, createAlways)
	if err != nil {
		return err
	}

	for offset := 0; offset < len(data); {
		end := offset + 32*1024
		if end > len(data) {
			end = len(data)
		}
		var written uint32
		if err := syscall.WriteFile(handle, data[offset:end], &written, nil); err != nil {
			return cleanupTemp(tempPath, handle, err)
		}
		if written == 0 {
			return cleanupTemp(tempPath, handle, io.ErrShortWrite)
		}
		offset += int(written)
	}
	if err := syscall.FlushFileBuffers(handle); err != nil {
		return cleanupTemp(tempPath, handle, err)
	}
	if err := syscall.CloseHandle(handle); err != nil {
		return cleanupTemp(tempPath, syscall.InvalidHandle, err)
	}

	from, err := syscall.UTF16PtrFromString(tempPath)
	if err != nil {
		return cleanupTemp(tempPath, syscall.InvalidHandle, err)
	}
	to, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return cleanupTemp(tempPath, syscall.InvalidHandle, err)
	}
	result, _, callErr := moveFileExProc.Call(
		uintptr(unsafe.Pointer(from)),
		uintptr(unsafe.Pointer(to)),
		moveFileReplaceExisting|moveFileWriteThrough,
	)
	if result == 0 {
		if callErr == syscall.Errno(0) {
			callErr = syscall.GetLastError()
		}
		return cleanupTemp(tempPath, syscall.InvalidHandle, callErr)
	}
	return nil
}

func openFile(path string, access, creation uint32) (syscall.Handle, error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	handle, err := syscall.CreateFile(
		name,
		access,
		syscall.FILE_SHARE_READ,
		nil,
		creation,
		fileAttributeNormal,
		0,
	)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	if handle == syscall.InvalidHandle {
		return syscall.InvalidHandle, syscall.GetLastError()
	}
	return handle, nil
}

func closeWithError(handle syscall.Handle, err error) error {
	if closeErr := syscall.CloseHandle(handle); closeErr != nil {
		return errors.Join(err, fmt.Errorf("close snapshot: %w", closeErr))
	}
	return err
}

func cleanupTemp(path string, handle syscall.Handle, cause error) error {
	if handle != syscall.InvalidHandle {
		if err := syscall.CloseHandle(handle); err != nil {
			cause = errors.Join(cause, fmt.Errorf("close temporary snapshot: %w", err))
		}
	}
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return errors.Join(cause, err)
	}
	if result, _, callErr := deleteFileProc.Call(uintptr(unsafe.Pointer(name))); result == 0 {
		if callErr == syscall.Errno(0) {
			callErr = syscall.GetLastError()
		}
		if !errors.Is(callErr, syscall.ERROR_FILE_NOT_FOUND) {
			cause = errors.Join(cause, fmt.Errorf("remove temporary snapshot: %w", callErr))
		}
	}
	return cause
}
