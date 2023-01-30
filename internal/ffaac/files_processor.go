package ffaac

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/utilitywarehouse/finance-fulfilment-archive-api/pkg/pb/bfaa"
	"golang.org/x/sync/errgroup"
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

	wg, ctx := errgroup.WithContext(parentCtx)

	wg.Go(func() error {
		return p.filesFinder.Run(ctx, fileCh)
	})

	for i := 0; i < p.workers; i++ {
		w := &fileSaverWorker{
			faaClient: p.archiveAPIClient,
			fileChan:  fileCh,
			basedir:   p.basedir,
		}
		wg.Go(func() error {
			return w.Run(ctx)
		})
	}

	if err := wg.Wait(); err != nil {
		return err
	}

	logrus.Infof("Processing ended")
	return nil
}
