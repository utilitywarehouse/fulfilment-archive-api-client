package ffaac

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
)

type FileFinder struct {
	basedir   string
	outChan   chan<- string
	errCh     chan<- error
	recursive bool
}

func NewFileFinder(basedir string, fileChan chan<- string, recursive bool, errCh chan<- error) *FileFinder {
	return &FileFinder{basedir: basedir, outChan: fileChan, recursive: recursive, errCh: errCh}
}

func (f *FileFinder) Run(ctx context.Context) {
	defer close(f.outChan)

	f.findRecursive(ctx, f.basedir)
}

func (f *FileFinder) findRecursive(ctx context.Context, dir string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		f.errCh <- errors.Wrapf(err, "Error listing files in dir %s", dir)
		return
	}

	for _, file := range files {
		fullFn := filepath.Join(dir, file.Name())
		if file.IsDir() && f.recursive {
			f.findRecursive(ctx, fullFn)
		}
		f.outChan <- fullFn
	}
}
