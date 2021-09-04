package ydfs

import (
	"os"
	"testing"
)

func Test_Read(t *testing.T) {
	fs, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	file, err := fs.Open("/go.mod")
	if err != nil {
		t.Error(err)
	}
	buf := make([]byte, 120)
	n, err := file.Read(buf)
	t.Logf("n = %v, err = %v\ndata: %v", n, err, string(buf))
	stats, err := file.Stat()
	if err != nil {
		t.Error(err)
	}
	t.Log(stats.Name(), stats.Size(), stats.IsDir())
}
