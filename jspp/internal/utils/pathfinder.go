package utils

import (
	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/kernel"
	lxApp "github.com/epicoon/lxgo/kernel/app"
)

type pathfinder struct {
	*lxApp.Pathfinder

	pp jspp.IPreprocessor
}

func NewPathfinder(pp jspp.IPreprocessor) kernel.IPathfinder {
	return &pathfinder{
		Pathfinder: lxApp.NewPathfinder(pp.App().Pathfinder().GetRoot()),
		pp:         pp,
	}
}

func (pf *pathfinder) GetAbsPath(path string) string {

	if path[0] == '{' {
		//TODO

	}

	//TODO

	return pf.pp.App().Pathfinder().GetAbsPath(path)
}
