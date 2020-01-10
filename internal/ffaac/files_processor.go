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
	workers          int
	filesFinder      FilesFinder
}

func NewFileProcessor(faaClient bfaa.BillFulfilmentArchiveAPIClient, basedir string, workers int, filesFinder FilesFinder) *FilesProcessor {
	return &FilesProcessor{
		archiveAPIClient: faaClient,
		basedir:          basedir,
		workers:          workers,
		filesFinder:      filesFinder,
	}
}

func (p *FilesProcessor) ProcessFiles(parentCtx context.Context) error {
	fileCh := make(chan string, 100)

	ctx, cancel := context.WithCancel(parentCtx)

	wg := sync.WaitGroup{}
	wg.Add(1)

	errorsCh := make(chan error, p.workers+1)

	go func() {
		if err := p.filesFinder.Run(ctx, fileCh); err != nil {
			errorsCh <- err
		}
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
			if err := w.Run(ctx); err != nil {
				errorsCh <- err
			}
			wg.Done()
		}()
	}

	var err error
	go func() {
		//	this will trigger either when a first worker has error, or when the error channel is closed.
		//	We need to cancel the context so that the workers will be stopped
		err = <-errorsCh
		cancel()
	}()

	wg.Wait()
	close(errorsCh)

	logrus.Infof("Processing ended")
	return err
}
