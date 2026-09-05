package online

import (
	"context"
	"io"
	"os"
	"sync"
	"time"
)

// ProgressiveFileStream wraps an open *os.File being written to concurrently by an external process.
// It implements io.ReadSeekCloser. When Read() encounters EOF while the writer is still active,
// it blocks until new data is flushed or the writer finishes.
type ProgressiveFileStream struct {
	mu         sync.Mutex
	file       *os.File
	writerDone <-chan struct{}
	writerErr  *error
	ctx        context.Context
	cancel     context.CancelFunc
	closed     bool
	onClose    func()
}

// NewProgressiveFileStream creates a new ProgressiveFileStream.
func NewProgressiveFileStream(
	ctx context.Context,
	cancel context.CancelFunc,
	file *os.File,
	writerDone <-chan struct{},
	writerErr *error,
	onClose func(),
) *ProgressiveFileStream {
	return &ProgressiveFileStream{
		file:       file,
		writerDone: writerDone,
		writerErr:  writerErr,
		ctx:        ctx,
		cancel:     cancel,
		onClose:    onClose,
	}
}

// Read reads bytes from the underlying file. If EOF is reached while the writer is still active,
// it waits for new bytes to be written.
func (p *ProgressiveFileStream) Read(b []byte) (int, error) {
	for {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return 0, io.ErrClosedPipe
		}
		n, err := p.file.Read(b)
		p.mu.Unlock()

		if n > 0 {
			return n, nil
		}
		if err != nil && err != io.EOF {
			return 0, err
		}

		// Check context cancellation
		if p.ctx != nil {
			select {
			case <-p.ctx.Done():
				return 0, p.ctx.Err()
			default:
			}
		}

		// Check if writer finished
		select {
		case <-p.writerDone:
			p.mu.Lock()
			if p.writerErr != nil && *p.writerErr != nil {
				err := *p.writerErr
				p.mu.Unlock()
				return 0, err
			}
			n, err := p.file.Read(b)
			p.mu.Unlock()
			return n, err
		case <-time.After(25 * time.Millisecond):
		}
	}
}

// Seek seeks to an offset in the underlying file.
func (p *ProgressiveFileStream) Seek(offset int64, whence int) (int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0, io.ErrClosedPipe
	}
	return p.file.Seek(offset, whence)
}

// Close closes the read file and triggers cancellation/cleanup.
func (p *ProgressiveFileStream) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	if p.cancel != nil {
		p.cancel()
	}
	if p.onClose != nil {
		p.onClose()
	}
	return p.file.Close()
}
