package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/epicoon/lxgo/jspp"
)

func BuildCore(pp jspp.IPreprocessor, ppRoot string, srcFlag bool) error {
	if err := buildClient(pp, ppRoot, srcFlag); err != nil {
		return err
	}

	if err := buildServer(pp, ppRoot); err != nil {
		return err
	}

	return nil
}

func buildClient(pp jspp.IPreprocessor, ppRoot string, srcFlag bool) error {
	coreSrc := filepath.Join(ppRoot, "js/src/core.js")
	compiler := pp.CompilerBuilder().
		SetClientContext().
		SetFilePath(coreSrc).
		Compiler()
	code, err := compiler.Run()
	if err != nil {
		return fmt.Errorf("can not compile core: %v", err)
	}

	var destPath string
	if srcFlag {
		destPath = filepath.Join(ppRoot, "js/build/core.js")
	} else {
		destPath = pp.App().Pathfinder().GetAbsPath(pp.Config().CorePath)
	}
	err = os.WriteFile(destPath, []byte(code), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}

	return nil
}

func buildServer(pp jspp.IPreprocessor, ppRoot string) error {
	coreSrc := filepath.Join(ppRoot, "js/src/core.js")
	compiler := pp.CompilerBuilder().
		SetServerContext().
		SetFilePath(coreSrc).
		Compiler()
	code, err := compiler.Run()
	if err != nil {
		return fmt.Errorf("can not compile core: %v", err)
	}

	destPath := pp.App().Pathfinder().GetAbsPath(pp.Config().CorePath)
	re := regexp.MustCompile(`\.js$`)
	destPath = re.ReplaceAllString(destPath, "-server.js")

	err = os.WriteFile(destPath, []byte(code), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}

	return nil
}
