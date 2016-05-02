package downloader

import (
	"os"
	"fmt"
	"io"
	"strings"
)

// TempDir is the dir that is used to temporarily store fragments
var TempDir = os.TempDir() + "/.go-dl"

// A Download represents an in-progress or completed download
type Download struct {
	file    *os.File
	done    chan error

	max     int
	current int
}

// File gets the output file of the download
func (d *Download) File() *os.File {
	return d.file
}

// Progress gets a float value from 0 - 1 depending on the progress of the download
func (d *Download) Progress() float32 {
	return float32(d.current) / float32(d.max)
}

// Wait blocks the thread until the download is complete
func (d *Download) Wait() error {
	for err := range d.done {
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Download) progress(i int) {
	d.current += i
}

type byteCounterWriter struct {
	writer         io.Writer
	onBytesWritten func(int)
}

func (w *byteCounterWriter) Write(p []byte) (n int, err error) {
	if w.onBytesWritten != nil {
		w.onBytesWritten(len(p))
	}
	return w.writer.Write(p)
}

func createDestFile(destFileName string) (*os.File, error) {
	origDest := destFileName

	lastSlash := strings.LastIndex(origDest, "/")
	if lastSlash == -1 {
		lastSlash = 0
	}

	lastDot := strings.LastIndex(origDest[lastSlash:], ".")

	for i := 1; ; i++ {
		if _, err := os.Stat(destFileName); err != nil {
			break
		} else {
			if lastDot == -1 {
				destFileName = fmt.Sprintf("%s (%d)", origDest, i)
			} else {
				destFileName = fmt.Sprintf("%s (%d).%s", origDest[:lastSlash + lastDot], i, origDest[lastSlash + lastDot + 1:])
			}
		}
	}

	return os.Create(destFileName)
}
