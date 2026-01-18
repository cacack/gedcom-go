package decoder

import "io"

// progressReader wraps an io.Reader to track bytes read and invoke a callback.
type progressReader struct {
	reader    io.Reader
	bytesRead int64
	totalSize int64
	callback  ProgressCallback
}

// Read implements io.Reader, tracking cumulative bytes and invoking the callback.
func (p *progressReader) Read(buf []byte) (n int, err error) {
	n, err = p.reader.Read(buf)
	if n > 0 {
		p.bytesRead += int64(n)
		total := p.totalSize
		if total == 0 {
			total = -1
		}
		p.callback(p.bytesRead, total)
	}
	return n, err
}
