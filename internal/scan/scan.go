package scan

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/haunterhx/hashx/internal/entropy"
)

func Analyze(path string, maxSize int64) Result {
	r := Result{Path: path}

	f, err := os.Open(path)
	if err != nil {
		r.Err = err.Error()
		return r
	}

	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		r.Err = err.Error()
		return r
	}

	r.Size = info.Size()
	if maxSize > 0 && r.Size > maxSize {
		r.Err = "skipped: exceeds the maximum size."
		return r
	}

	h := sha256.New()
	var hist [256]uint64
	buf := make([]byte, 64*1024)
	var total uint64

	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			h.Write(buf[:n])
			for _, b := range buf[:n] {
				hist[b]++
			}
			total += uint64(n)
		}

		if rerr == io.EOF {
			break
		}

		if rerr != nil {
			r.Err = rerr.Error()
			return r
		}
	}

	r.SHA256 = hex.EncodeToString(h.Sum(nil))
	r.Entropy = entropy.Shannon(hist, total)

	return r
}
