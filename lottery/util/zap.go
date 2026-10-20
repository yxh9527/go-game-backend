package util

import (
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func InitZapLogger(path, prefix string) {
	currentPath := ""
	if path == "" {
		curPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return
		}
		currentPath = curPath
	} else {
		currentPath = path
	}

	hook := lumberjack.Logger{
		Filename:   filepath.Join(currentPath, "logs/debug", "lottery.log"), // 日志文件路径
		MaxSize:    128,                                                     // 最大日志大小（Mb级别）
		MaxBackups: 30,                                                      // 最多保留30个备份
		MaxAge:     7,                                                       // days
		Compress:   true,                                                    // 是否压缩 disabled by default
		LocalTime:  true,
	}

	err_hook := lumberjack.Logger{
		Filename:   filepath.Join(currentPath, "logs/error", "lottery.log"), // 日志文件路径
		MaxSize:    128,                                                     // 最大日志大小（Mb级别）
		MaxBackups: 30,                                                      // 最多保留30个备份
		MaxAge:     7,                                                       // days
		Compress:   true,                                                    // 是否压缩 disabled by default
		LocalTime:  true,
	}

	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})

	fileDebugging := zapcore.AddSync(&hook)
	fileErrors := zapcore.AddSync(&err_hook)

	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)

	consoleCfg := zap.NewDevelopmentEncoderConfig()
	consoleCfg.EncodeTime = customTimeEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(consoleCfg)

	fileCfg := zap.NewProductionEncoderConfig()
	fileCfg.EncodeTime = customTimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(fileCfg)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		zapcore.NewCore(fileEncoder, fileErrors, highPriority),
		zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
		zapcore.NewCore(fileEncoder, fileDebugging, lowPriority),
	)

	logger := zap.New(core, zap.AddStacktrace(zap.WarnLevel))

	zap.ReplaceGlobals(logger)
}

func customTimeEncoder(time time.Time, encoder zapcore.PrimitiveArrayEncoder) {
	encoder.AppendString(time.Format("2006-01-02 15:04:05.000000"))
}
