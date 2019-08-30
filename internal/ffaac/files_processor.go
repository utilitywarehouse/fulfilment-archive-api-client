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
}

func NewFileProcessor(faaClient bfaa.BillFulfilmentArchiveAPIClient, basedir string, recursive bool, workers int) *FilesProcessor {
	return &FilesProcessor{
		archiveAPIClient: faaClient,
		basedir:          basedir,
		recursive:        recursive,
		workers:          workers,
	}
}

func (p *FilesProcessor) ProcessFiles(ctx context.Context) {
	fileCh := make(chan string, 100)
	errCh := make(chan error, 100)
	defer close(errCh)

	wg := sync.WaitGroup{}
	wg.Add(1)

	ff := &filesFinder{basedir: p.basedir, filesCh: fileCh, recursive: p.recursive, errCh: errCh}
	go func() {
		ff.Run(ctx)
		wg.Done()
	}()

	wg.Add(p.workers)
	for i := 0; i < p.workers; i++ {
		w := &fileSaverWorker{
			faaClient: p.archiveAPIClient,
			fileChan:  fileCh,
			errCh:     errCh,
			basedir:   p.basedir,
		}
		go func() {
			w.Run(ctx)
			wg.Done()
		}()
	}

	go func() {
		for err := range errCh {
			logrus.Error(err)
		}
	}()

	wg.Wait()
}
