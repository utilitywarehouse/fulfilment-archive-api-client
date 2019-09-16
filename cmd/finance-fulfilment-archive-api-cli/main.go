package main

import (
	"context"
	"math"
	"time"

	"github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/ffaac"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	cli "github.com/jawher/mow.cli"
	log "github.com/sirupsen/logrus"
	"github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/pb/bfaa"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"os"
	"os/signal"
	"strings"
	"syscall"
)

var version string // populated at compile time

const (
	appName = "finance-fulfilment-archive-api-cli"
	appDesc = "This application is used to upload items to finance-fulfilment-archive"
)

func main() {
	app := cli.App(appName, appDesc)

	fulfilmentArchAPIAddr := app.String(cli.StringOpt{
		Name:   "a fulfilment-archive-api-address",
		Desc:   "The address of fulfilment-archive-api gRPC service",
		EnvVar: "FULFILMENT_ARCHIVE_API_ADDRESS",
		Value:  "finance-fulfilment-archive-api:8090",
	})
	fulfilmentArchAPIgrpcLB := app.String(cli.StringOpt{
		Name:   "b fulfilment-archive-api-grpc-balancer",
		Desc:   "GRPC load balancer name for fulfilment archive API. Options: pick_first,round_robin,xds,grpclb",
		EnvVar: "FULFILMENT_ARCHIVE_API_GRPC_BALANCER",
		Value:  "round_robin",
	})

	logLevel := app.String(cli.StringOpt{
		Name:   "l log-level",
		Desc:   "log level [debug|info|warn|error]",
		EnvVar: "LOG_LEVEL",
		Value:  "info",
	})

	logFormat := app.String(cli.StringOpt{
		Name:   "f log-format",
		Desc:   "Log format, if set to text will use text as logging format, otherwise will use json",
		EnvVar: "LOG_FORMAT",
		Value:  "json",
	})

	workers := app.Int(cli.IntOpt{
		Name:   "w workers",
		Desc:   "The number of workers to use for uploading in parallel",
		EnvVar: "WORKERS",
		Value:  10,
	})

	recursive := app.Bool(cli.BoolOpt{
		Name:   "r recursive",
		Desc:   "Upload recursively all the files in the specified folder",
		EnvVar: "RECURSIVE",
		Value:  true,
	})

	basedir := app.String(cli.StringArg{
		Name:   "BASEDIR",
		Desc:   "The base directory where to upload all the files from",
		EnvVar: "BASEDIR",
	})

	fileExtensions := app.String(cli.StringOpt{
		Name:   "e file-extensions",
		Desc:   "The list of file extensions to process",
		EnvVar: "FILE_EXTENSIONS",
		Value:  "pdf,csv",
	})

	app.Action = func() {
		configureLogger(*logLevel, *logFormat)

		ctx, cancel := context.WithCancel(context.Background())

		fulfilmentArchAPIConn := initialiseGRPCClientConnection(ctx, fulfilmentArchAPIAddr, fulfilmentArchAPIgrpcLB)
		defer func() {
			if err := fulfilmentArchAPIConn.Close(); err != nil {
				log.WithError(err).Error("error while shutting down fulfilment archive api connection")
			}
		}()

		faaClient := bfaa.NewBillFulfilmentArchiveAPIClient(fulfilmentArchAPIConn)

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		doneCh := make(chan bool)

		log.Infof("finance-fulfilment-archive-api-cli version: %s", version)

		filesProcessor := ffaac.NewFileProcessor(faaClient, *basedir, *recursive, *workers, strings.Split(*fileExtensions, ","))
		go func() {
			filesProcessor.ProcessFiles(ctx)
			doneCh <- true
		}()

		select {
		case <-sigChan:
			cancel()
		case <-doneCh:
			cancel()
			return
		}
	}

	if err := app.Run(os.Args); err != nil {
		log.WithError(err).Panic("unable to run app")
	}
}

func configureLogger(level, format string) {
	l, err := log.ParseLevel(level)
	if err != nil {
		log.WithFields(log.Fields{"log_level": level}).
			WithError(err).
			Panic("invalid log level")
	}
	log.SetLevel(l)

	format = strings.ToLower(format)
	if format != "text" && format != "json" {
		log.Panicf("invalid log format: %s", format)
	}
	if format == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	}
	log.SetOutput(os.Stdout)
}

func initialiseGRPCClientConnection(ctx context.Context, grpcClientAddress *string, grpcLoadBalancer *string) *grpc.ClientConn {
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt32),
			grpc.MaxCallSendMsgSize(math.MaxInt32),
		),
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(
			[]grpc_retry.CallOption{
				grpc_retry.WithBackoff(grpc_retry.BackoffLinearWithJitter(100*time.Millisecond, 0.1)),
				grpc_retry.WithMax(3),
				grpc_retry.WithCodes(codes.Unknown, codes.DeadlineExceeded, codes.Internal, codes.Unavailable),
			}...,
		)),
	}
	if grpcLoadBalancer != nil {
		opts = append(opts, grpc.WithBalancerName(*grpcLoadBalancer))
	}

	grpcClientConn, err := grpc.DialContext(ctx, *grpcClientAddress, opts...)

	if err != nil {
		log.WithFields(log.Fields{"grpc_client_address": *grpcClientAddress}).
			WithError(err).
			Panic("grpc client connection failed")
	}
	return grpcClientConn
}
