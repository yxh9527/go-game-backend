package common

import (
	"os"
	"path/filepath"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitZapLogger() {
	currentPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return
	}
	hook := lumberjack.Logger{
		Filename:   filepath.Join(currentPath, "logs", "client-api.log"),
		MaxSize:    128,
		MaxBackups: 30,
		MaxAge:     7,
		Compress:   true,
		LocalTime:  true,
	}
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})
	consoleCfg := zap.NewDevelopmentEncoderConfig()
	consoleCfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}
	fileCfg := zap.NewProductionEncoderConfig()
	fileCfg.EncodeTime = consoleCfg.EncodeTime
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewConsoleEncoder(consoleCfg), zapcore.Lock(os.Stderr), highPriority),
		zapcore.NewCore(zapcore.NewJSONEncoder(fileCfg), zapcore.AddSync(&hook), highPriority),
		zapcore.NewCore(zapcore.NewConsoleEncoder(consoleCfg), zapcore.Lock(os.Stdout), lowPriority),
		zapcore.NewCore(zapcore.NewJSONEncoder(fileCfg), zapcore.AddSync(&hook), lowPriority),
	)
	zap.ReplaceGlobals(zap.New(core, zap.AddStacktrace(zap.WarnLevel)))
}
