package singleinstance

import (
	"fmt"
	"os"
)

func writePID(file *os.File) error {
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}
	_, err := fmt.Fprintf(file, "%d\n", os.Getpid())
	return err
}
