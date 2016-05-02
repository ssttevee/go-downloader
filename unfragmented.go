package downloader

import (
	"io"
)

// UnfragmentedDownload starts downloading the file directly to the destination and without fragmenting the file
//
// Returns a Download instance
func (d *Downloader) UnfragmentedDownload(destFileName string) (*Download, error) {
	res, err := d.httpClient().Get(d.url)
	if err != nil {
		return nil, err
	}

	// create the destination file
	out, err := createDestFile(destFileName)
	if err != nil {
		return nil, err
	}

	dl := d.makeDownload()
	dl.file = out

	go func() {
		if _, err := io.Copy(&byteCounterWriter{
			&byteCounterWriter{out, d.OnBytesReceived},
			dl.progress,
		}, res.Body); err != nil {
			dl.done <- err
		}

		dl.file.Close()

		close(dl.done)
	}()

	return dl, nil
}