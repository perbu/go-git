//go:build !darwin && !linux

package mmap

import (
	"errors"
	"io"

	"github.com/go-git/go-billy/v6"

	"github.com/go-git/go-git/v6/plumbing"
)

type PackScanner struct{}

func NewPackScanner(hashSize int, pack, idx, rev billy.File) (*PackScanner, error) {
	return nil, errors.New("pack scanner is only supported in linux or darwin")
}

func NewPackScannerRaw(hashSize int, pack, idx billy.File) (*PackScanner, error) {
	return nil, errors.New("pack scanner is only supported in linux or darwin")
}

func (s *PackScanner) GetRawCompressed(h plumbing.Hash) (plumbing.ObjectType, int64, io.ReadCloser, error) {
	return 0, 0, nil, errors.New("pack scanner is only supported in linux or darwin")
}

func (s *PackScanner) Close() error {
	return nil
}
