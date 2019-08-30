package ffaac_test

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/ffaac"
	"github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/ffaac/mocks"
	"github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/pb/bfaa"
)

//go:generate mockgen -package=mocks -destination=mocks/bill_fulfilment_archive_api.go github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/pb/bfaa BillFulfilmentArchiveAPIClient

const workers = 10

type processorTestInstances struct {
	ctrl                 *gomock.Controller
	mockArchiveAPIClient *mocks.MockBillFulfilmentArchiveAPIClient

	basedir   string
	processor *ffaac.FilesProcessor
}

func initProcessorMocks(t *testing.T, recursive bool) processorTestInstances {
	ctrl := gomock.NewController(t)
	ti := processorTestInstances{
		ctrl:                 ctrl,
		mockArchiveAPIClient: mocks.NewMockBillFulfilmentArchiveAPIClient(ctrl),
	}
	rootPath, err := ioutil.TempDir("", "processor-test")
	require.NoError(t, err)

	ti.basedir = rootPath

	ti.processor = ffaac.NewFileProcessor(ti.mockArchiveAPIClient, ti.basedir, recursive, workers)
	return ti
}

func (ti *processorTestInstances) finish() {
	ti.ctrl.Finish()
	if err := os.RemoveAll(ti.basedir); err != nil {
		logrus.Error(err)
	}
}

func TestSimpleDir(t *testing.T) {
	ti := initProcessorMocks(t, true)
	fileNames := []string{"one.pdf", "two.pdf"}
	ti.createTestFiles(t, fileNames...)

	ctx := context.Background()
	for _, fileName := range fileNames {
		ti.mockArchiveAPIClient.EXPECT().SaveBillFulfilmentArchive(ctx, getExpectedSaveRequest(fileName))
	}

	ti.processor.ProcessFiles(ctx)
}

func getExpectedSaveRequest(fileName string) *bfaa.SaveBillFulfilmentArchiveRequest {
	return &bfaa.SaveBillFulfilmentArchiveRequest{
		Id:      fileName,
		Archive: &bfaa.BillFulfilmentArchive{Data: []byte(fileName)},
	}
}

func (ti *processorTestInstances) createTestFiles(t *testing.T, files ...string) {
	for _, fileName := range files {
		fullFn := filepath.Join(ti.basedir, fileName)
		err := ioutil.WriteFile(fullFn, []byte(fileName), 0666)
		require.NoError(t, err)
	}
}
