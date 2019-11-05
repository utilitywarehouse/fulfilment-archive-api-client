package ffaac_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

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

func initProcessorMocks(t *testing.T, recursive bool, fileExtensions ...string) processorTestInstances {
	ctrl := gomock.NewController(t)
	ti := processorTestInstances{
		ctrl:                 ctrl,
		mockArchiveAPIClient: mocks.NewMockBillFulfilmentArchiveAPIClient(ctrl),
	}
	rootPath, err := ioutil.TempDir("", "processor-test")
	require.NoError(t, err)

	ti.basedir = rootPath

	filesFinder := ffaac.NewFilesFinder(ti.basedir, recursive, fileExtensions)
	ti.processor = ffaac.NewFileProcessor(ti.mockArchiveAPIClient, ti.basedir, workers, filesFinder)
	return ti
}

func (ti *processorTestInstances) finish() {
	ti.ctrl.Finish()
	if err := os.RemoveAll(ti.basedir); err != nil {
		logrus.Error(err)
	}
}

func TestProcessEmptyDir(t *testing.T) {
	ti := initProcessorMocks(t, true)
	defer ti.finish()

	ti.mockArchiveAPIClient.EXPECT().SaveBillFulfilmentArchive(gomock.Any(), gomock.Any()).Times(0)
	err := ti.processor.ProcessFiles(context.Background())
	assert.NoError(t, err)
}

func TestProcessSimpleDir(t *testing.T) {
	ti := initProcessorMocks(t, true, "pdf")
	defer ti.finish()

	fileNames := []string{"one.pdf", "two.pdf"}
	ti.createTestFiles(t, fileNames...)

	for _, fileName := range fileNames {
		ti.mockArchiveAPIClient.EXPECT().SaveBillFulfilmentArchive(gomock.Any(), getExpectedSaveRequest(fileName)).Return(nil, nil).Times(1)
	}

	err := ti.processor.ProcessFiles(context.Background())
	assert.NoError(t, err)
}

func TestProcessStopOnError(t *testing.T) {
	ti := initProcessorMocks(t, true, "pdf", "csv")
	defer ti.finish()

	fileNames := []string{"one.pdf", "two.pdf", "three.csv"}
	ti.createTestFiles(t, fileNames...)

	err := errors.New("dummy error")
	//	error on the first file. The other two should not be processed any more
	ti.mockArchiveAPIClient.EXPECT().SaveBillFulfilmentArchive(gomock.Any(), getExpectedSaveRequest("one.pdf")).Return(nil, err).Times(1)
	ti.mockArchiveAPIClient.EXPECT().SaveBillFulfilmentArchive(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	expErr := ti.processor.ProcessFiles(context.Background())
	assert.Error(t, expErr)
	assert.True(t, errors.Is(expErr, err))
}

func TestProcessWithChildDirsRecursive(t *testing.T) {
	ti := initProcessorMocks(t, true, "pdf")
	defer ti.finish()

	fileNames := []string{"one.pdf", "two.pdf",
		filepath.Join("fold1", "thee.pdf"),
		filepath.Join("fold1", "fold2", "four.pdf")}
	ti.createTestFiles(t, fileNames...)

	for _, fileName := range fileNames {
		ti.mockArchiveAPIClient.EXPECT().SaveBillFulfilmentArchive(gomock.Any(), getExpectedSaveRequest(fileName)).Return(nil, nil).Times(1)
	}

	err := ti.processor.ProcessFiles(context.Background())
	assert.NoError(t, err)

}

func TestProcessManyFilesRecursive(t *testing.T) {
	ti := initProcessorMocks(t, true, "pdf")
	defer ti.finish()

	var allFileNames []string
	for i := 0; i < 500; i++ {
		allFileNames = append(allFileNames, fmt.Sprintf("file%d.pdf", i))
	}
	for i := 0; i < 500; i++ {
		allFileNames = append(allFileNames, filepath.Join("fold1", fmt.Sprintf("file%d.pdf", i)))
	}
	for i := 0; i < 500; i++ {
		allFileNames = append(allFileNames, filepath.Join("fold1", "fold2", fmt.Sprintf("file%d.pdf", i)))
	}
	for i := 0; i < 500; i++ {
		allFileNames = append(allFileNames, filepath.Join("fold1", "fold3", fmt.Sprintf("file%d.pdf", i)))
	}

	ti.createTestFiles(t, allFileNames...)

	for _, fileName := range allFileNames {
		ti.mockArchiveAPIClient.EXPECT().SaveBillFulfilmentArchive(gomock.Any(), getExpectedSaveRequest(fileName)).Return(nil, nil).Times(1)
	}

	err := ti.processor.ProcessFiles(context.Background())
	assert.NoError(t, err)

}

func TestProcessWithChildDirsNonRecursive(t *testing.T) {
	ti := initProcessorMocks(t, false, "pdf")
	defer ti.finish()

	baseFileNames := []string{"one.pdf", "two.pdf"}
	childFileNames := []string{
		filepath.Join("fold1", "thee.pdf"),
		filepath.Join("fold1", "fold2", "four.pdf")}
	allFiles := append(baseFileNames, childFileNames...)
	ti.createTestFiles(t, allFiles...)

	for _, fileName := range baseFileNames {
		ti.mockArchiveAPIClient.EXPECT().SaveBillFulfilmentArchive(gomock.Any(), getExpectedSaveRequest(fileName)).Return(nil, nil).Times(1)
	}

	err := ti.processor.ProcessFiles(context.Background())
	assert.NoError(t, err)

}

func TestProcessSkipNotIncludedFiles(t *testing.T) {
	ti := initProcessorMocks(t, true, "csv")
	defer ti.finish()

	includedFiles := []string{"one.csv", filepath.Join("fold1", "thee.csv")}
	excludedFiles := []string{
		"two.pdf",
		filepath.Join("fold1", "thee.pdf"),
		filepath.Join("fold1", "fold2", "four.pdf")}
	allFiles := append(includedFiles, excludedFiles...)
	ti.createTestFiles(t, allFiles...)

	for _, fileName := range includedFiles {
		ti.mockArchiveAPIClient.EXPECT().SaveBillFulfilmentArchive(gomock.Any(), getExpectedSaveRequest(fileName)).Return(nil, nil).Times(1)
	}

	err := ti.processor.ProcessFiles(context.Background())
	assert.NoError(t, err)

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
		fileDir := filepath.Dir(fullFn)
		err := os.MkdirAll(fileDir, 0777)
		require.NoError(t, err)

		err = ioutil.WriteFile(fullFn, []byte(fileName), 0666)
		require.NoError(t, err)
	}
}
