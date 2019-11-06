package ffaac

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

type FilesFinder interface {
	Run(ctx context.Context, filesCh chan<- string)
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

func (f *filesFinder) Run(ctx context.Context, filesCh chan<- string) {
	defer close(filesCh)

	f.findRecursive(ctx, f.basedir, "", filesCh)
}

func (f *filesFinder) findRecursive(ctx context.Context, dir string, baseRelativeDir string, filesCh chan<- string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		logrus.WithError(err).Errorf("Error listing files in dir %s", dir)
		return
	}

	for _, file := range files {
		fullFn := filepath.Join(dir, file.Name())
		baseRelativeName := filepath.Join(baseRelativeDir, file.Name())
		if file.IsDir() {
			if f.recursive {
				f.findRecursive(ctx, fullFn, baseRelativeName, filesCh)
			}
		} else {
			if f.isFileIncluded(baseRelativeName) {
				filesCh <- baseRelativeName
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
