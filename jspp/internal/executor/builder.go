package executor

import (
	"github.com/epicoon/lxgo/jspp"
)

type executorBuilder struct {
	executor *executor
}

func Builder() jspp.IJSExecutorBuilder {
	return &executorBuilder{
		executor: &executor{},
	}
}

func (b *executorBuilder) Executor() jspp.IJSExecutor {
	return b.executor
}

func (b *executorBuilder) SetPreprocessor(pp jspp.IPreprocessor) jspp.IJSExecutorBuilder {
	b.executor.pp = pp
	return b
}

func (b *executorBuilder) SetCode(code string) jspp.IJSExecutorBuilder {
	b.executor.code = code
	return b
}
