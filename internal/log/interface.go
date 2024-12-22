package log

type Logger interface {
	WithField(string, interface{}) Logger
	WithError(error) Logger

	Info(string)
	Warn(string)
	Success(string)
	Fail(string)
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Successf(string, ...interface{})
	Failf(string, ...interface{})

	IncreaseIndent()
	DecreaseIndent()
	ResetIndent()
}
