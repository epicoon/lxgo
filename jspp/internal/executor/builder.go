package executor

import (
	"github.com/epicoon/lxgo/jspp"
)

type executorBuilder struct {
	executor *executor
}

func Builder() jspp.IExecutorBuilder {
	return &executorBuilder{
		executor: &executor{},
	}
}

func (b *executorBuilder) Executor() jspp.IExecutor {
	return b.executor
}

func (b *executorBuilder) SetPreprocessor(pp jspp.IPreprocessor) jspp.IExecutorBuilder {
	b.executor.pp = pp
	return b
}

func (b *executorBuilder) SetCode(code string) jspp.IExecutorBuilder {
	b.executor.code = code
	return b
}
