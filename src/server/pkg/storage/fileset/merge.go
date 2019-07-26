package fileset

import (
	"fmt"
	"io"
	"strings"

	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/index"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/tar"
)

type stream interface {
	next() error
	key() string
}

type mergeFunc func([]stream) error

type fileStream struct {
	r   *Reader
	hdr *index.Header
}

func (fs *fileStream) next() error {
	var err error
	fs.hdr, err = fs.r.Next()
	return err
}

func (fs *fileStream) key() string {
	return fs.hdr.Hdr.Name
}

func idxMergeFunc(w *Writer) mergeFunc {
	return func(ss []stream) error {
		// (bryce) this will implement an index merge, which will be used by the distributed merge process.
		return nil
	}
}

func contentMergeFunc(w *Writer) mergeFunc {
	return func(ss []stream) error {
		// Convert generic streams to file streams.
		var fileStreams []*fileStream
		for _, s := range ss {
			fileStreams = append(fileStreams, s.(*fileStream))
		}
		// Setup tag streams for tag merge.
		var tagStreams []stream
		var size int64
		for _, fs := range fileStreams {
			tagStreams = append(tagStreams, &tagStream{
				r: fs.r,
				// (bryce) header tag removed by first next call.
				tags: fs.hdr.Idx.DataOp.Tags,
			})
			size += fs.hdr.Idx.SizeBytes
		}
		// Write header for file.
		hdr := &index.Header{
			Hdr: &tar.Header{
				Name: fileStreams[0].hdr.Hdr.Name,
				Size: size,
			},
		}
		if err := w.WriteHeader(hdr); err != nil {
			return err
		}
		// Merge file content.
		return merge(tagStreams, tagMergeFunc(w))
	}
}

type tagStream struct {
	r    *Reader
	tags []*index.Tag
}

func (ts *tagStream) next() error {
	ts.tags = ts.tags[1:]
	if len(ts.tags) == 0 {
		return io.EOF
	}
	return nil
}

func (ts *tagStream) key() string {
	return ts.tags[0].Id
}

func tagMergeFunc(w *Writer) mergeFunc {
	return func(ss []stream) error {
		// (bryce) this should be an Internal error type.
		if len(ss) > 1 {
			return fmt.Errorf("tags should be distinct within a file")
		}
		// Convert generic stream to tag stream.
		tagStream := ss[0].(*tagStream)
		// Copy tagged data to writer.
		w.StartTag(tagStream.tags[0].Id)
		return CopyN(w, tagStream.r, tagStream.tags[0].SizeBytes)
	}
}

type mergePriorityQueue struct {
	queue []stream
	f     mergeFunc
	size  int
}

func (mq *mergePriorityQueue) key(i int) string {
	return mq.queue[i].key()
}

func (mq *mergePriorityQueue) insert(s stream) error {
	// Get next in stream and insert it.
	if err := s.next(); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	mq.queue[mq.size+1] = s
	mq.size++
	// Propagate insert up the queue
	i := mq.size
	for i > 1 {
		if strings.Compare(mq.key(i/2), mq.key(i)) <= 0 {
			break
		}
		mq.swap(i/2, i)
		i /= 2
	}
	return nil
}

func (mq *mergePriorityQueue) next() []stream {
	ss := []stream{mq.queue[1]}
	mq.fill()
	// Keep popping streams off the queue if they have the same key.
	for mq.queue[1] != nil && strings.Compare(mq.key(1), ss[0].key()) == 0 {
		ss = append(ss, mq.queue[1])
		mq.fill()
	}
	return ss
}

func (mq *mergePriorityQueue) fill() {
	// Replace first stream with last
	mq.queue[1] = mq.queue[mq.size]
	mq.queue[mq.size] = nil
	mq.size--
	// Propagate last stream down the queue
	i := 1
	var next int
	for {
		left, right := i*2, i*2+1
		if left > mq.size {
			break
		} else if right > mq.size || strings.Compare(mq.key(left), mq.key(right)) <= 0 {
			next = left
		} else {
			next = right
		}
		if strings.Compare(mq.key(i), mq.key(next)) <= 0 {
			break
		}
		mq.swap(i, next)
		i = next
	}
}

func (mq *mergePriorityQueue) swap(i, j int) {
	mq.queue[i], mq.queue[j] = mq.queue[j], mq.queue[i]
}

func merge(ss []stream, f func([]stream) error) error {
	if len(ss) == 0 {
		return nil
	}
	mq := &mergePriorityQueue{queue: make([]stream, len(ss)+1)}
	// Insert streams.
	for _, s := range ss {
		if err := mq.insert(s); err != nil {
			return err
		}
	}
	for mq.queue[1] != nil {
		// Get next streams and merge them.
		ss := mq.next()
		if err := f(ss); err != nil {
			return err
		}
		// Re-insert streams
		for _, s := range ss {
			if err := mq.insert(s); err != nil {
				return err
			}
		}
	}
	return nil
}
