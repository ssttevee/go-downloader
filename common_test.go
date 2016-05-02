package downloader

import (
	"os"
	"github.com/dustin/go-humanize"
	"testing"
	"runtime"
)

const (
	testUrl = "https://storage.googleapis.com/ssttevee/misc/ts3_recording_15_12_23_23_43_55.wav"
	testSize = 1699280
	testFragSize = 1 << 20 // 1MiB
	testFilename = "test.wav"
)

func makeDownload() *Downloader {
	return &Downloader{
		url: testUrl,
		size: testSize,
		fragmentable: true,
		FragmentSize: testFragSize,
	}
}

func analyze(t *testing.T, file *os.File) {
	finfo, err := os.Stat(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s file size is %v", finfo.Name(), humanize.Bytes(uint64(finfo.Size())))

	if uint64(finfo.Size()) == uint64(testSize) {
		t.Logf("Expected file is %v", humanize.Bytes(uint64(testSize)))
	} else {
		t.Fatalf("Expected file is %v", humanize.Bytes(uint64(testSize)))
	}

	t.Logf("Deleting %s", finfo.Name())

	if err := os.Remove(finfo.Name()); err != nil {
		t.Fatal(err)
	}
}

func testDownload(t *testing.T, d *Download) {
	const fmtStr = "Download Progress: %d out of %d - %.2f"

	lastProgress := float32(-1)
	for p := float32(0); p < float32(1); p = d.Percent() {
		if lastProgress != p {
			t.Logf(fmtStr, d.Current(), d.TotalBytes(), p * float32(100))
			lastProgress = p
		}

		runtime.Gosched()
	}

	t.Logf(fmtStr, d.Current(), d.TotalBytes(), d.Percent() * float32(100))

	if err := d.Wait(); err != nil {
		t.Fatal(err)
	}

	d.File().Close()

	t.Logf("Saved to %s", d.File().Name())

	analyze(t, d.File())
}
