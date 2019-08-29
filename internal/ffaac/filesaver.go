package ffaac

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"

	"github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/pb/bfaa"
)

type FileSaver struct {
	faaClient bfaa.BillFulfilmentArchiveAPIClient
	fileChan  <-chan string
	errCh     chan<- error
}

func NewFileSaver(faaClient bfaa.BillFulfilmentArchiveAPIClient, fileChan <-chan string, errCh chan<- error) *FileSaver {
	return &FileSaver{
		faaClient: faaClient,
		fileChan:  fileChan,
		errCh:     errCh,
	}
}

func (f *FileSaver) Run(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case fn, ok := <-f.fileChan:
		if ok {
			if err := f.saveFile(ctx, fn); err != nil {
				f.errCh <- err
			}
		}
	}
}

func (f *FileSaver) saveFile(ctx context.Context, fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", fileName)
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return errors.Wrapf(err, "failed reading bytes for file %s", fileName)
	}

	_, err = f.faaClient.SaveBillFulfilmentArchive(ctx, &bfaa.SaveBillFulfilmentArchiveRequest{
		Id:      fileName,
		Archive: &bfaa.BillFulfilmentArchive{Data: bytes},
	})
	if err != nil {
		return errors.Wrapf(err, "failed calling the fulfilment archive api for file %s", fileName)
	}
	return nil
}
