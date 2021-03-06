/*
Asynchronously download files via HTTP

Able to use multiple threads and avoid filename collisions
 */
package downloader

import (
	"net/http"
	"strconv"
	"math"
)

const defaultFragmentSize = int64(1 << 22) // 4MiB

// A Downloader encapsulates limited remote file info and helps to download it
type Downloader struct {
	url             string
	size            int64

	fragmentable    bool
	FragmentSize    int64

	HttpClient      *http.Client

	OnBytesReceived func(int)

	dirty           bool
}

// Url returns the target url
func (d *Downloader) Url() string {
	return d.url
}

// Url returns the size of the file at the target url
func (d *Downloader) Size() int64 {
	return d.size
}

// Fragmentable returns whether or not the target url supports the `Range` header
func (d *Downloader) Fragmentable() bool {
	return d.fragmentable
}

// NumFragments calculates the number of fragments for the target url
func (d *Downloader) NumFragments() int {
	if (!d.fragmentable) {
		return 1
	}
	return int(math.Ceil(float64(d.size) / float64(d.FragmentSize)))
}

func (d *Downloader) httpClient() *http.Client {
	client := d.HttpClient
	if client == nil {
		client = http.DefaultClient
	}

	return client
}

func (d *Downloader) makeDownload() *Download {
	return &Download{
		done: make(chan error),
		max: int(d.size),
	}
}

// New tries to get the `Content-Length` of the target url
// and also checks for the `Accept-Ranges` header for fragmentability
//
// Returns an instance of Downloader on success
func New(url string) (*Downloader, error) {
	res, err := http.Head(url)
	if err != nil {
		return nil, err
	}

	length, err := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return nil, err
	}

	fragmentable := res.Header.Get("Accept-Ranges") == "bytes"

	return &Downloader{
		url: url,
		size: length,
		fragmentable: fragmentable,
		FragmentSize: defaultFragmentSize,
	}, nil
}
