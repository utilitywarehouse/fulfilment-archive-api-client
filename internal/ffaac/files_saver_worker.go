package ffaac

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/pb/bfaa"
)

type fileSaverWorker struct {
	faaClient bfaa.BillFulfilmentArchiveAPIClient
	fileChan  <-chan string
	basedir   string
}

func (f *fileSaverWorker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case fn, ok := <-f.fileChan:
			if ok {
				if err := f.sendFileToArchiveAPI(ctx, fn); err != nil {
					logrus.Error(err)
				}
			} else {
				return
			}
		}
	}
}

func (f *fileSaverWorker) sendFileToArchiveAPI(ctx context.Context, fileName string) error {
	logrus.Infof("Processing file %s", fileName)
	file, err := os.Open(filepath.Join(f.basedir, fileName))
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", fileName, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logrus.WithError(err).Errorf("failed closing file %s", fileName)
		}
	}()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed reading bytes for file %s: %w", fileName, err)
	}

	_, err = f.faaClient.SaveBillFulfilmentArchive(ctx, &bfaa.SaveBillFulfilmentArchiveRequest{
		Id:      fileName,
		Archive: &bfaa.BillFulfilmentArchive{Data: bytes},
	})
	if err != nil {
		return fmt.Errorf("failed calling the fulfilment archive api for file %s: %w", fileName, err)
	}
	return nil
}
