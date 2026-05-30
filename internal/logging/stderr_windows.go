package logging

import (
	"os"

	"golang.org/x/sys/windows"
)

// RedirectStderr points the OS-level stderr handle at f so that runtime
// panics and CGO crashes are captured even after FreeConsole() is called.
func RedirectStderr(f *os.File) {
	_ = windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(f.Fd()))
}