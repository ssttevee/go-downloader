package downloader

import (
	"os"
	"fmt"
	"net/http"
	"io"
	"strings"
)

// TempDir is the dir that is used to temporarily store fragments
var TempDir = os.TempDir() + "/.go-dl"

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

type dlJob struct {
	index     int
	byteStart int64
	byteEnd   int64
}

// Download starts downloading the file to the destination with some number of threads
//
// Returns the number of bytes downloaded
func (d *Downloader) Download(destFileName string, threads int) (int64, error) {
	// make sure the target url is fragmentable
	// otherwise redirect to UnfragmentedDownload
	if !d.fragmentable {
		return d.UnfragmentedDownload(destFileName)
	}

	// make sure there is at least one thread
	if threads < 1 {
		threads = 1
	}

	// make sure the temp dir exists
	os.Mkdir(TempDir, os.ModeDir)

	// get the number of fragments
	fragCount := d.NumFragments()

	// create an array of file paths
	// don't use *os.File because it's bytes can't be read without being reopened anyways
	fragmentPaths := make([]string, fragCount)

	// prepare job and return channels
	jobsChan := make(chan dlJob, fragCount)
	errsChan := make(chan error)

	// prepare a var to keep track of downloaded bytes
	totalBytesDownloaded := int64(0)

	// setup workers
	for i := 0; i < threads; i++ {
		go func() {
			for job := range jobsChan {
				tempPath := fmt.Sprintf("%s/%s.part%d", TempDir, destFileName, job.index)

				// check if the fragment already exists
				if finfo, err := os.Stat(tempPath); err == nil {
					// assume that same size is synonymous with same file
					if finfo.Size() == job.byteEnd - job.byteStart + 1 {
						// call OnBytesReceived so that we'll end up with 100% when we finish
						d.OnBytesReceived(int(finfo.Size()))

						// add the path the the array
						fragmentPaths[job.index] = tempPath
						errsChan <- nil
						continue
					}
				}

				f, err := os.Create(tempPath)
				if err != nil {
					errsChan <- err
					return
				}

				req, err := http.NewRequest("GET", d.url, nil)
				if err != nil {
					errsChan <- err
					return
				}

				// set the `Range` header for the request to only download a fragment
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", job.byteStart, job.byteEnd))

				res, err := d.httpClient().Do(req)
				if err != nil {
					errsChan <- err
					return
				}

				if bytesDownloaded, err := io.Copy(&byteCounterWriter{
					writer: f,
					onBytesWritten: d.OnBytesReceived,
				}, res.Body); err != nil {
					errsChan <- err
					return
				} else {
					totalBytesDownloaded += bytesDownloaded
				}

				f.Close()

				fragmentPaths[job.index] = f.Name()
				errsChan <- nil
			}
		}()
	}

	// create the download jobs
	for i := range fragmentPaths {
		job := dlJob{index: i}

		job.byteStart = int64(i) * d.FragmentSize
		if i == fragCount - 1 {
			job.byteEnd = d.size - 1
		} else {
			job.byteEnd = int64(i + 1) * d.FragmentSize - 1
		}

		jobsChan <- job
	}

	// wait for all the jobs to finish
	for range fragmentPaths {
		if err := <-errsChan; err != nil {
			return 0, err
		}
	}

	// create the destination file
	out, err := createDestFile(destFileName)
	if err != nil {
		return 0, err
	}

	for _, fname := range fragmentPaths {
		// open the fragment so we can read it
		f, err := os.Open(fname)
		if err != nil {
			return 0, err
		}

		// copy all the bytes from the fragment to the destination file
		if _, err := io.Copy(out, f); err != nil {
			return 0, err
		}

		// close the file so we can't delete it
		f.Close()

		// delete the fragment file
		os.Remove(f.Name())
	}

	out.Close()

	return totalBytesDownloaded, nil
}

// UnfragmentedDownload starts downloading the file to the destination without fragmenting the file
//
// Returns the number of bytes downloaded
func (d *Downloader) UnfragmentedDownload(destFileName string) (int64, error) {
	res, err := d.httpClient().Get(d.url)
	if err != nil {
		return 0, err
	}

	// create the destination file
	out, err := createDestFile(destFileName)
	if err != nil {
		return 0, err
	}

	return io.Copy(out, res.Body)
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
