// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logger

import (
	"log"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	LogContainer     logContainer
	loggerInit       sync.Once
	simpleLoggerInit sync.Once
)

type logContainer struct {
	logger       *zap.Logger
	simpleLogger *zap.SugaredLogger
}

// GetLogger returns the pointer to the logger and creates one if none exists
func (l *logContainer) GetLogger() *zap.Logger {
	loggerInit.Do(func() {
		l.logger = zap.New(getCombinedCore())
	})
	return l.logger
}

// GetSimpleLogger returns the pointer to the sugared logger and creates one
// if none exists
func (l *logContainer) GetSimpleLogger() *zap.SugaredLogger {
	simpleLoggerInit.Do(func() {
		logger := zap.New(getCombinedCore())
		l.simpleLogger = logger.Sugar()
	})
	return l.simpleLogger
}

// String mirrors zap.String
func (l *logContainer) String(key string, val string) zap.Field {
	return zap.String(key, val)
}

// Int mirrors zap.Int
func (l *logContainer) Int(key string, val int) zap.Field {
	return zap.Int(key, val)
}

func getConsoleEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getJsonEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.EpochTimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

//TODO make this work in addition to stdout
func getLogWriter() zapcore.WriteSyncer {
	f, err := os.Create("/tmp/u-bmc.log")
	if err != nil {
		log.Fatalf("unable to create logfile: %v", err)
	}
	return zapcore.AddSync(f)
}

func getConsoleCore() zapcore.Core {
	return zapcore.NewCore(getConsoleEncoder(), zapcore.AddSync(os.Stdout), zapcore.InfoLevel)
}

func getJsonCore() zapcore.Core {
	return zapcore.NewCore(getJsonEncoder(), getLogWriter(), zapcore.InfoLevel)
}

func getCombinedCore() zapcore.Core {
	return zapcore.NewTee(getConsoleCore(), getJsonCore())
}
