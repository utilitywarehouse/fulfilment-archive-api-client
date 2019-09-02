package ffaac

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/pb/bfaa"
)

type FilesProcessor struct {
	archiveAPIClient bfaa.BillFulfilmentArchiveAPIClient
	basedir          string
	recursive        bool
	workers          int
	fileExtensions   []string
}

func NewFileProcessor(faaClient bfaa.BillFulfilmentArchiveAPIClient, basedir string, recursive bool, workers int, fileExtensions []string) *FilesProcessor {
	return &FilesProcessor{
		archiveAPIClient: faaClient,
		basedir:          basedir,
		recursive:        recursive,
		workers:          workers,
		fileExtensions:   fileExtensions,
	}
}

func (p *FilesProcessor) ProcessFiles(ctx context.Context) {
	logrus.Infof("Starting processing files in %s. Recursive: %v. Looking for files with extensions: %v", p.basedir, p.recursive, p.fileExtensions)

	fileCh := make(chan string, 100)

	wg := sync.WaitGroup{}
	wg.Add(1)

	ff := &filesFinder{basedir: p.basedir, filesCh: fileCh, recursive: p.recursive, fileExtensions: p.fileExtensions}
	go func() {
		ff.Run(ctx)
		wg.Done()
	}()

	wg.Add(p.workers)
	for i := 0; i < p.workers; i++ {
		w := &fileSaverWorker{
			faaClient: p.archiveAPIClient,
			fileChan:  fileCh,
			basedir:   p.basedir,
		}
		go func() {
			w.Run(ctx)
			wg.Done()
		}()
	}

	wg.Wait()
	logrus.Infof("Processing ended")
}
