package ffaac_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/ffaac"
	"github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/ffaac/mocks"
	"github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/pb/bfaa"
)

//go:generate mockgen -package=mocks -destination=mocks/bill_fulfilment_archive_api.go github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/pb/bfaa BillFulfilmentArchiveAPIClient
//go:generate mockgen -package=mocks -destination=mocks/files_finder.go github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/ffaac FilesFinder

const workers = 10

type processorTestInstances struct {
	ctrl                 *gomock.Controller
	mockArchiveAPIClient *mocks.MockBillFulfilmentArchiveAPIClient

	basedir         string
	processor       *ffaac.FilesProcessor
	mockFilesFinder *mocks.MockFilesFinder
}

func initProcessorWithRealFinder(t *testing.T, recursive bool, fileExtensions ...string) processorTestInstances {
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

func initProcessorWithMockFinder(t *testing.T) processorTestInstances {
	ctrl := gomock.NewController(t)
	ti := processorTestInstances{
		ctrl:                 ctrl,
		mockArchiveAPIClient: mocks.NewMockBillFulfilmentArchiveAPIClient(ctrl),
		mockFilesFinder:      mocks.NewMockFilesFinder(ctrl),
	}

	rootPath, err := ioutil.TempDir("", "processor-test")
	require.NoError(t, err)

	ti.basedir = rootPath

	ti.processor = ffaac.NewFileProcessor(ti.mockArchiveAPIClient, ti.basedir, workers, ti.mockFilesFinder)
	return ti
}

func (ti *processorTestInstances) finish() {
	ti.ctrl.Finish()
	if err := os.RemoveAll(ti.basedir); err != nil {
		logrus.Error(err)
	}
}

func TestProcessEmptyDir(t *testing.T) {
	ti := initProcessorWithRealFinder(t, true)
	defer ti.finish()

	ti.mockArchiveAPIClient.EXPECT().SaveBillFulfilmentArchive(gomock.Any(), gomock.Any()).Times(0)
	err := ti.processor.ProcessFiles(context.Background())
	assert.NoError(t, err)
}

func TestProcessNotExistingDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ti := processorTestInstances{
		ctrl:                 ctrl,
		mockArchiveAPIClient: mocks.NewMockBillFulfilmentArchiveAPIClient(ctrl),
	}

	ti.basedir = "some-not-existing-dir"

	filesFinder := ffaac.NewFilesFinder(ti.basedir, true, nil)
	ti.processor = ffaac.NewFileProcessor(ti.mockArchiveAPIClient, ti.basedir, workers, filesFinder)

	ti.mockArchiveAPIClient.EXPECT().SaveBillFulfilmentArchive(gomock.Any(), gomock.Any()).Times(0)
	err := ti.processor.ProcessFiles(context.Background())
	assert.Error(t, err)
}

func TestProcessSimpleDir(t *testing.T) {
	ti := initProcessorWithRealFinder(t, true, "pdf")
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
	ti := initProcessorWithMockFinder(t)
	defer ti.finish()

	fileNames := []string{"one.pdf", "two.pdf", "three.csv"}
	ti.createTestFiles(t, fileNames...)

	errorSent := make(chan struct{})
	err := errors.New("dummy error")
	ti.mockFilesFinder.EXPECT().Run(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, filesCh chan<- string) error {
			filesCh <- "one.pdf"
			/*	block until the error is triggered by SaveBillFulfilmentArchive, and wait more so that the error is processed */
			<-errorSent
			time.Sleep(500 * time.Millisecond)
			/*	more sends should not trigger other saves as the workers should be done by now */
			filesCh <- "two.pdf"
			filesCh <- "three.pdf"
			// wait more so that those sends should be processed
			time.Sleep(500 * time.Millisecond)
			close(filesCh)
			return nil
		})

	saveCalled := 0
	ti.mockArchiveAPIClient.EXPECT().SaveBillFulfilmentArchive(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, in *bfaa.SaveBillFulfilmentArchiveRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
			saveCalled++
			if saveCalled == 1 { // only on first call return error
				close(errorSent)
				return nil, err
			}

			return nil, nil
		})

	expErr := ti.processor.ProcessFiles(context.Background())
	assert.Error(t, expErr)
	assert.True(t, errors.Is(expErr, err))
	assert.Equal(t, 1, saveCalled)
}

func TestProcessWithChildDirsRecursive(t *testing.T) {
	ti := initProcessorWithRealFinder(t, true, "pdf")
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
	ti := initProcessorWithRealFinder(t, true, "pdf")
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
	ti := initProcessorWithRealFinder(t, false, "pdf")
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
	ti := initProcessorWithRealFinder(t, true, "csv")
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
