package downloader

import (
	"testing"
	"os"
)

func TestCreateDestFile(t *testing.T) {
	names := map[string][]string {
		"something": {
			"something",
			"something (1)",
			"something (2)",
		},
		"somethingelse.mp4": {
			"somethingelse.mp4",
			"somethingelse (1).mp4",
			"somethingelse (2).mp4",
		},
		"some/thing/over/here": {
			"some/thing/over/here",
			"some/thing/over/here (1)",
			"some/thing/over/here (2)",
		},
		"some/thing/over/there.wav": {
			"some/thing/over/there.wav",
			"some/thing/over/there (1).wav",
			"some/thing/over/there (2).wav",
		},
	}

	os.MkdirAll("some/thing/over", os.ModeDir)

	for name, results := range names {
		files := make([]*os.File, 3)

		for i, result := range results {
			file, err := createDestFile(name)
			if err != nil {
				t.Error(err)
			}

			t.Log("created destination file is", file.Name())

			file.Close()

			if file.Name() != result {
				t.Error("expected", result, "got", file.Name())
			}

			files[i] = file
		}

		for _, file := range files {
			os.Remove(file.Name())
		}
	}

	os.RemoveAll("some")
}
