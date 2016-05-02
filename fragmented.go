package downloader

import (
	"os"
	"io"
	"fmt"
	"net/http"
)

type dlJob struct {
	index     int
	byteStart int64
	byteEnd   int64
}

type dlRes struct {
	index int
	err   error
	path  string
}

// Download starts downloading the file to the destination with some number of threads
//
// Returns a Download instance
func (d *Downloader) Download(destFileName string, threads int) (*Download, error) {
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

	// prepare the return variable
	dl := d.makeDownload()

	// get the number of fragments
	frags := d.NumFragments()

	// prepare job and return channels
	jobsChan := make(chan dlJob, frags)
	resChan := make(chan dlRes, frags)

	// prepare a var to keep track of downloaded bytes
	totalBytesDownloaded := int64(0)

	// setup workers
	for i := 0; i < threads; i++ {
		go func() {
			for job := range jobsChan {
				tempPath := fmt.Sprintf("%s/%s.part%d", TempDir, destFileName, job.index)

				ret := dlRes{index: job.index}

				// check if the fragment already exists
				if finfo, err := os.Stat(tempPath); err == nil {
					// assume that same size is synonymous with same file
					if finfo.Size() == job.byteEnd - job.byteStart + 1 {
						// call OnBytesReceived so that we'll end up with 100% when we finish
						if d.OnBytesReceived != nil {
							d.OnBytesReceived(int(finfo.Size()))
						}
						dl.progress(int(finfo.Size()))

						ret.path = tempPath
						resChan <- ret
						continue
					}
				}

				f, err := os.Create(tempPath)
				if err != nil {
					ret.err = err
					resChan <- ret
					return
				}

				req, err := http.NewRequest("GET", d.url, nil)
				if err != nil {
					ret.err = err
					resChan <- ret
					return
				}

				// set the `Range` header for the request to only download a fragment
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", job.byteStart, job.byteEnd))

				res, err := d.httpClient().Do(req)
				if err != nil {
					ret.err = err
					resChan <- ret
					return
				}

				if bytesDownloaded, err := io.Copy(&byteCounterWriter{
					writer: &byteCounterWriter{f, d.OnBytesReceived},
					onBytesWritten: dl.progress,
				}, res.Body); err != nil {
					ret.err = err
					resChan <- ret
					return
				} else {
					totalBytesDownloaded += bytesDownloaded
				}

				f.Close()

				ret.path = f.Name()
				resChan <- ret
			}
		}()
	}

	// start finish listener
	go func() {
		// wait for all the jobs to finish
		for {
			if len(resChan) >= frags {
				close(resChan)
				break
			}
		}

		resCount := 0
		paths := make([]string, frags)

		// retrieve all the paths from the results
		for res := range resChan {
			if res.err != nil {
				dl.done <- res.err
				return
			}

			paths[res.index] = res.path
			resCount++

			if resCount == frags {
				break
			}
		}

		// create the destination file
		out, err := createDestFile(destFileName)
		if err != nil {
			dl.done <- err
			return
		}
		defer out.Close()

		dl.file = out

		// copy all parts to the output file
		for _, path := range paths {
			f, err := os.Open(path)
			if err != nil {
				dl.done <- err
				return
			}

			_, err = io.Copy(dl.file, f)
			if err != nil {
				dl.done <- err
				return
			}

			f.Close()

			if !d.dirty {
				os.Remove(path)
			}
		}

		close(dl.done)
	}()

	// create the download jobs
	for i := 0; i < frags; i++ {
		job := dlJob{index: i}

		job.byteStart = int64(i) * d.FragmentSize
		if i == frags - 1 {
			job.byteEnd = d.size - 1
		} else {
			job.byteEnd = int64(i + 1) * d.FragmentSize - 1
		}

		jobsChan <- job
	}

	close(jobsChan)

	return dl, nil
}