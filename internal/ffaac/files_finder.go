package ffaac

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type filesFinder struct {
	basedir        string
	filesCh        chan<- string
	errCh          chan<- error
	recursive      bool
	fileExtensions []string
}

func (f *filesFinder) Run(ctx context.Context) {
	defer close(f.filesCh)

	f.findRecursive(ctx, f.basedir, "")
}

func (f *filesFinder) findRecursive(ctx context.Context, dir string, baseRelativeDir string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		f.errCh <- errors.Wrapf(err, "Error listing files in dir %s", dir)
		return
	}

	for _, file := range files {
		fullFn := filepath.Join(dir, file.Name())
		baseRelativeName := filepath.Join(baseRelativeDir, file.Name())
		if file.IsDir() {
			if f.recursive {
				f.findRecursive(ctx, fullFn, baseRelativeName)
			}
		} else {
			if f.isFileIncluded(baseRelativeName) {
				f.filesCh <- baseRelativeName
			}
		}
	}
}

func (f *filesFinder) isFileIncluded(fileName string) bool {
	for _, extension := range f.fileExtensions {
		if strings.HasSuffix(fileName, extension) {
			return true
		}
	}
	return false
}
