package logger

import "go.uber.org/zap"

var Log *zap.SugaredLogger

func Init() {
	Log = zap.Must(zap.NewProduction()).Sugar()
}

func Sync() {
	_ = Log.Sync()
}

