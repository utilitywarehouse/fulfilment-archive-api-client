# Finance fulfilment archive api CLI
Command line utility that saves files through the fulfilment-archive-api service.
It processes all the files in a folder recursively and does this in a parallel fashion, by using multiple workers.

## Setup

To install, run go get, then follow build instructions from the new directory.

```bash
go get github.com/utilitywarehouse/finance-fulfilment-archive-api-cli
cd  $GOPATH/src/github.com/utilitywarehouse/finance-fulfilment-archive-api-cli
```

### Dependencies

* Fulfilment Archive API

### Usage

#### Options

```bash
Usage: finance-fulfilment-archive-api-cli [OPTIONS] BASEDIR

This application is used to upload items to finance-fulfilment-archive

Arguments:                                     
  BASEDIR                                      The base directory where to upload all the files from (env $BASEDIR)
                                               
Options:                                       
  -a, --fulfilment-archive-api-address         The address of fulfilment-archive-api gRPC service (env $FULFILMENT_ARCHIVE_API_ADDRESS) (default "finance-fulfilment-archive-api:8090")
  -b, --fulfilment-archive-api-grpc-balancer   GRPC load balancer name for fulfilment archive API. Options: pick_first,round_robin,xds,grpclb (env $FULFILMENT_ARCHIVE_API_GRPC_BALANCER) (default "round_robin")
  -l, --log-level                              log level [debug|info|warn|error] (env $LOG_LEVEL) (default "info")
  -f, --log-format                             Log format, if set to text will use text as logging format, otherwise will use json (env $LOG_FORMAT) (default "json")
  -w, --workers                                The number of workers to use for uploading in parallel (env $WORKERS) (default 10)
  -r, --recursive                              Upload recursively all the files in the specified folder (env $RECURSIVE) (default true)
  -e, --file-extensions                        The list of file extensions to process (env $FILE_EXTENSIONS) (default "pdf,csv")
```

## Building

```bash
make all 
```

## Testing

```bash
make test
```