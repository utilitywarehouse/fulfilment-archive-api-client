package ffaac

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type FilesFinder interface {
	Run(ctx context.Context, filesCh chan<- string) error
}

func NewFilesFinder(basedir string, recursive bool, fileExtensions []string,
) FilesFinder {
	return &filesFinder{
		basedir:        basedir,
		recursive:      recursive,
		fileExtensions: fileExtensions,
	}
}

type filesFinder struct {
	basedir        string
	recursive      bool
	fileExtensions []string
}

func (f *filesFinder) Run(ctx context.Context, filesCh chan<- string) error {
	defer close(filesCh)

	if err := f.findRecursive(ctx, f.basedir, "", filesCh); err != nil {
		return err
	}
	return nil
}

func (f *filesFinder) findRecursive(ctx context.Context, dir string, baseRelativeDir string, filesCh chan<- string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed listing files in dir %s: %w", dir, err)
	}

	for _, file := range files {
		fullFn := filepath.Join(dir, file.Name())
		baseRelativeName := filepath.Join(baseRelativeDir, file.Name())
		if file.IsDir() {
			if f.recursive {
				if err := f.findRecursive(ctx, fullFn, baseRelativeName, filesCh); err != nil {
					return err
				}
			}
		} else {
			if f.isFileIncluded(baseRelativeName) {
				filesCh <- baseRelativeName
			}
		}
	}
	return nil
}

func (f *filesFinder) isFileIncluded(fileName string) bool {
	for _, extension := range f.fileExtensions {
		if strings.HasSuffix(fileName, extension) {
			return true
		}
	}
	return false
}
