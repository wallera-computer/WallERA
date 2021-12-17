package log

import "go.uber.org/zap"

func Production(_ ...zap.Option) *zap.Logger {
	return zap.NewNop()
}

func Development(opts ...zap.Option) *zap.Logger {
	opts = append(opts, zap.WithCaller(true))
	l, err := zap.NewDevelopment(
		opts...,
	)

	if err != nil {
		panic(err)
	}

	return l
}
