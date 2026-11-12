// Package ctxcheck holds what the packages that accept a Context share
// (issue #743): the rule that a nil Context is never cancelled, a document
// copy that checks the context per record, and a test helper that cancels at
// an exact check.
package ctxcheck

import (
	"context"
	"sync"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Err returns ctx.Err(), or nil when ctx is nil.
func Err(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

// Clone is d.Clone() checking ctx before each record: once ctx is done it
// stops and returns ctx's error and no document. A nil d is cloned to nil.
// Everything but the records is copied by Document.Clone itself; the record
// loop mirrors Document.Clone's, and TestClone pins that the two agree.
func Clone(ctx context.Context, d *gedcom.Document) (*gedcom.Document, error) {
	if err := Err(ctx); err != nil {
		return nil, err
	}
	if d == nil {
		return nil, nil
	}
	shell := *d
	shell.Records, shell.XRefMap = nil, nil
	copied := shell.Clone()
	copied.Records = make([]*gedcom.Record, len(d.Records))
	for i, record := range d.Records {
		if err := Err(ctx); err != nil {
			return nil, err
		}
		c := record.Clone()
		copied.Records[i] = c
		if c != nil && c.XRef != "" {
			copied.XRefMap[c.XRef] = c
		}
	}
	return copied, nil
}

// CancelAfter returns a Context whose Err reports nil for its first n calls
// and context.Canceled from then on. It lets a test cancel an operation at an
// exact check, to show the operation checks where it claims to; it is not for
// library code. Its Done channel is nil (never closed): Err is what the
// library polls.
func CancelAfter(n int64) context.Context {
	return &countdown{Context: context.Background(), left: n}
}

type countdown struct {
	context.Context
	left int64
	mu   sync.Mutex
}

func (c *countdown) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.left <= 0 {
		return context.Canceled
	}
	c.left--
	return nil
}
