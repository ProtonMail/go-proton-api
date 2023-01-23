package proton

import (
	"bytes"
	"context"
	"github.com/bradenaw/juniper/parallel"
)

// AttachmentAllocator abstract the attachment download buffer creation.
type AttachmentAllocator interface {
	// NewBuffer should return a new byte buffer for use. Note that this function may be called from multiple go-routines.
	NewBuffer() *bytes.Buffer
}

type DefaultAttachmentAllocator struct{}

func NewDefaultAttachmentAllocator() *DefaultAttachmentAllocator {
	return &DefaultAttachmentAllocator{}
}

func (DefaultAttachmentAllocator) NewBuffer() *bytes.Buffer {
	return bytes.NewBuffer(nil)
}

// Scheduler allows the user to specify how the attachment data for the message should be downloaded.
type Scheduler interface {
	Schedule(ctx context.Context, attachmentIDs []string, storageProvider AttachmentAllocator, downloader func(string, *bytes.Buffer) error) ([]*bytes.Buffer, error)
}

// SequentialScheduler downloads the attachments one by one.
type SequentialScheduler struct{}

func NewSequentialScheduler() *SequentialScheduler {
	return &SequentialScheduler{}
}

func (SequentialScheduler) Schedule(ctx context.Context, attachmentIDs []string, storageProvider AttachmentAllocator, downloader func(string, *bytes.Buffer) error) ([]*bytes.Buffer, error) {
	result := make([]*bytes.Buffer, len(attachmentIDs))
	for i, v := range attachmentIDs {

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		buffer := storageProvider.NewBuffer()
		if err := downloader(v, buffer); err != nil {
			return nil, err
		}

		result[i] = buffer
	}

	return result, nil
}

type ParallelScheduler struct {
	workers int
}

func NewParallelScheduler(workers int) *ParallelScheduler {
	return &ParallelScheduler{workers: workers}
}

func (p ParallelScheduler) Schedule(ctx context.Context, attachmentIDs []string, storageProvider AttachmentAllocator, downloader func(string, *bytes.Buffer) error) ([]*bytes.Buffer, error) {
	return parallel.MapContext(ctx, p.workers, attachmentIDs, func(ctx context.Context, id string) (*bytes.Buffer, error) {
		buffer := storageProvider.NewBuffer()
		if err := downloader(id, buffer); err != nil {
			return nil, err
		}

		return buffer, nil
	})

}
