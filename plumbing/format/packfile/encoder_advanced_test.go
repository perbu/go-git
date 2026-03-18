package packfile_test

import (
	"bytes"
	"io"
	"math/rand"
	"testing"

	"github.com/go-git/go-billy/v6/memfs"
	fixtures "github.com/go-git/go-git-fixtures/v5"
	"github.com/stretchr/testify/suite"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/cache"
	"github.com/go-git/go-git/v6/plumbing/format/idxfile"
	. "github.com/go-git/go-git/v6/plumbing/format/packfile"
	"github.com/go-git/go-git/v6/plumbing/storer"
	"github.com/go-git/go-git/v6/storage/filesystem"
)

type EncoderAdvancedSuite struct {
	suite.Suite
}

func TestEncoderAdvancedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(EncoderAdvancedSuite))
}

func (s *EncoderAdvancedSuite) TestEncodeDecode() {
	if testing.Short() {
		s.T().Skip("skipping test in short mode.")
	}

	fixs := fixtures.Basic().ByTag("packfile").ByTag(".git")
	fixs = append(fixs, fixtures.ByURL("https://github.com/src-d/go-git.git").
		ByTag("packfile").ByTag(".git").One())

	for _, f := range fixs {
		storage := filesystem.NewStorage(f.DotGit(), cache.NewObjectLRUDefault())
		s.testEncodeDecode(storage, 10)
	}
}

func (s *EncoderAdvancedSuite) TestEncodeDecodeNoDeltaCompression() {
	if testing.Short() {
		s.T().Skip("skipping test in short mode.")
	}

	fixs := fixtures.Basic().ByTag("packfile").ByTag(".git")
	fixs = append(fixs, fixtures.ByURL("https://github.com/src-d/go-git.git").
		ByTag("packfile").ByTag(".git").One())

	for _, f := range fixs {
		storage := filesystem.NewStorage(f.DotGit(), cache.NewObjectLRUDefault())
		s.testEncodeDecode(storage, 0)
	}
}

// BenchmarkEncodeFromPackfile benchmarks encoding from a packfile-backed
// filesystem storage, which exercises the raw-copy optimization for packWindow=0.
func BenchmarkEncodeFromPackfile(b *testing.B) {
	fixs := fixtures.Basic().ByTag("packfile").ByTag(".git")
	f := fixs.One()
	storage := filesystem.NewStorage(f.DotGit(), cache.NewObjectLRUDefault())

	objIter, err := storage.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		b.Fatal(err)
	}

	var hashes []plumbing.Hash
	_ = objIter.ForEach(func(o plumbing.EncodedObject) error {
		hashes = append(hashes, o.Hash())
		return nil
	})

	b.Run("delta10", func(b *testing.B) {
		var totalBytes int64
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(nil)
			enc := NewEncoder(buf, storage, false)
			if _, err := enc.Encode(hashes, 10); err != nil {
				b.Fatal(err)
			}
			totalBytes = int64(buf.Len())
		}
		b.StopTimer()
		b.ReportMetric(float64(totalBytes), "output-bytes")
		b.ReportMetric(float64(len(hashes)), "objects")
	})

	b.Run("delta0", func(b *testing.B) {
		var totalBytes int64
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(nil)
			enc := NewEncoder(buf, storage, false)
			if _, err := enc.Encode(hashes, 0); err != nil {
				b.Fatal(err)
			}
			totalBytes = int64(buf.Len())
		}
		b.StopTimer()
		b.ReportMetric(float64(totalBytes), "output-bytes")
		b.ReportMetric(float64(len(hashes)), "objects")
	})
}

func (s *EncoderAdvancedSuite) testEncodeDecode(
	storage storer.Storer,
	packWindow uint,
) {
	objIter, err := storage.IterEncodedObjects(plumbing.AnyObject)
	s.NoError(err)

	expectedObjects := map[plumbing.Hash]bool{}
	var hashes []plumbing.Hash
	err = objIter.ForEach(func(o plumbing.EncodedObject) error {
		expectedObjects[o.Hash()] = true
		hashes = append(hashes, o.Hash())
		return err
	})
	s.NoError(err)

	// Shuffle hashes to avoid delta selector getting order right just because
	// the initial order is correct.
	auxHashes := make([]plumbing.Hash, len(hashes))
	for i, j := range rand.Perm(len(hashes)) {
		auxHashes[j] = hashes[i]
	}
	hashes = auxHashes

	buf := bytes.NewBuffer(nil)
	enc := NewEncoder(buf, storage, false)
	encodeHash, err := enc.Encode(hashes, packWindow)
	s.NoError(err)

	fs := memfs.New()
	f, err := fs.Create("packfile")
	s.NoError(err)

	_, err = f.Write(buf.Bytes())
	s.NoError(err)

	_, err = f.Seek(0, io.SeekStart)
	s.NoError(err)

	w := new(idxfile.Writer)
	parser := NewParser(NewScanner(f), WithScannerObservers(w))

	_, err = parser.Parse()
	s.NoError(err)
	index, err := w.Index()
	s.NoError(err)

	_, err = f.Seek(0, io.SeekStart)
	s.NoError(err)

	p := NewPackfile(f, WithIdx(index), WithFs(fs))

	decodeHash, err := p.ID()
	s.NoError(err)
	s.Equal(decodeHash, encodeHash)

	objIter, err = p.GetAll()
	s.NoError(err)
	obtainedObjects := map[plumbing.Hash]bool{}
	err = objIter.ForEach(func(o plumbing.EncodedObject) error {
		obtainedObjects[o.Hash()] = true
		return nil
	})
	s.NoError(err)
	s.Equal(expectedObjects, obtainedObjects)

	for h := range obtainedObjects {
		if !expectedObjects[h] {
			s.T().Errorf("obtained unexpected object: %s", h)
		}
	}

	for h := range expectedObjects {
		if !obtainedObjects[h] {
			s.T().Errorf("missing object: %s", h)
		}
	}
}
