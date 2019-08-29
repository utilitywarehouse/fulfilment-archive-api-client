package ffaac

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
)

type filesFinder struct {
	basedir   string
	filesCh   chan<- string
	errCh     chan<- error
	recursive bool
}

func (f *filesFinder) Run(ctx context.Context) {
	defer close(f.filesCh)

	f.findRecursive(ctx, f.basedir)
}

func (f *filesFinder) findRecursive(ctx context.Context, dir string) {
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
		f.filesCh <- fullFn
	}
}
