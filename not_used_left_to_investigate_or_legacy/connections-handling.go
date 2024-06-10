package main

import (
	"bufio"
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/network"
	"go.uber.org/zap"
	log "go.uber.org/zap"
)

func HandleNewConnections(ctx context.Context, logger *log.Logger, connsChan <-chan network.Conn) {
	for conn := range connsChan {
		logger.Info("handling new connection")
		stream, err := conn.NewStream(ctx)
		if err != nil {
			logger.Error("failed to open stream", zap.Error(err))
		}

		r := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
		go handleIncomingStream(logger, r)
	}
}

func handleStream(logger *log.Logger) network.StreamHandler {
	return func(stream network.Stream) {
		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
		go writeData(logger, rw.Writer)
		go readData(logger, rw.Reader)
	}
}

func handleIncomingStream(logger *log.Logger, rw *bufio.ReadWriter) {
	go writeData(logger, rw.Writer)
	go readData(logger, rw.Reader)
}

func readData(logger *log.Logger, rw *bufio.Reader) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			logger.Error("failed reading from stream", zap.Error(err))
		}

		if str == "" {
			return
		}
		if str != "\n" {
			logger.Info("received message", zap.String("message", str))
		}
	}
}

func writeData(logger *log.Logger, rw *bufio.Writer) {
	for {
		_, err := rw.WriteString(fmt.Sprintf("%s\n", "x"))
		if err != nil {
			logger.Error("failed writing to stream", zap.Error(err))
		}
		err = rw.Flush()
		if err != nil {
			logger.Error("failed flushing the stream", zap.Error(err))
		}
	}
}
