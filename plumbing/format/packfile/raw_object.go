package packfile

import (
	"io"

	"github.com/go-git/go-git/v6/plumbing"
)

// rawObject is a lightweight EncodedObject that holds only metadata
// (type, size, hash) without actual content. It is used with the
// raw-copy optimization where compressed bytes are copied directly
// from a source packfile, so the object content is never materialized.
type rawObject struct {
	typ  plumbing.ObjectType
	sz   int64
	hash plumbing.Hash
}

func (o *rawObject) Hash() plumbing.Hash               { return o.hash }
func (o *rawObject) Type() plumbing.ObjectType          { return o.typ }
func (o *rawObject) SetType(t plumbing.ObjectType)      { o.typ = t }
func (o *rawObject) Size() int64                        { return o.sz }
func (o *rawObject) SetSize(s int64)                    { o.sz = s }
func (o *rawObject) Reader() (io.ReadCloser, error)     { return io.NopCloser(io.LimitReader(nil, 0)), nil }
func (o *rawObject) Writer() (io.WriteCloser, error)    { return nil, nil }
