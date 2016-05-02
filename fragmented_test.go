package downloader

import (
	"testing"
	"net/http"
)

func TestDownloader_Download_All(t *testing.T) {
	for _, fraggable := range []bool{true, false} {
		for _, httpClient := range []*http.Client{nil, http.DefaultClient} {
			for _, bytesCallback := range []func(int){nil, func(int) {/* do nothing */}} {
				for _, threads := range []int{0, 1} {
					for _, dirty := range []bool{false, true} {
						downloader := makeDownload()
						downloader.fragmentable = fraggable
						downloader.HttpClient = httpClient
						downloader.OnBytesReceived = bytesCallback
						downloader.dirty = dirty

						d, err := downloader.Download("test.wav", threads)
						if err != nil {
							t.Fatal(err)
						}

						testDownload(t, d)
					}
				}
			}
		}
	}
}