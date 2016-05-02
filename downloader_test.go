package downloader

import "testing"

func TestNew(t *testing.T) {
	d, err := New(testUrl)
	if err != nil {
		t.Fatal(err)
	}

	if d.Url() != testUrl {
		t.Fatal("Expected url to be %s, got %s", testUrl, d.Url())
	}

	if d.Size() != testSize {
		t.Fatal("Expected size is %d, got %d", testSize, d.Size())
	}

	if !d.Fragmentable() {
		t.Fatal("Expected to be fragmented, got false")
	}
}

func TestNew_NonExistent(t *testing.T) {
	_, err := New("http://exists.not")
	if err == nil {
		t.Fatal("Expected error")
	}

	t.Log(err)
}
