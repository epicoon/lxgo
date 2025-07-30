package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/dop251/goja"
	"github.com/epicoon/lxgo/jspp"
)

type executor struct {
	pp   jspp.IPreprocessor
	code string
}

var _ jspp.IExecutor = (*executor)(nil)

type execResult struct {
	log    map[string][]string
	errors map[string][]string
	dumps  []string
	result any
	fatal  string
}

var _ jspp.IExecResult = (*execResult)(nil)

func (er *execResult) Log() map[string][]string {
	return er.log
}

func (er *execResult) Errors() map[string][]string {
	return er.errors
}

func (er *execResult) Dumps() []string {
	return er.dumps
}

func (er *execResult) Result() any {
	return er.result
}

func (er *execResult) Fatal() string {
	return er.fatal
}

type codePart struct {
	index int
	code  string
}

func (e *executor) Exec() (jspp.IExecResult, error) {
	vm := goja.New()

	c := make(chan codePart, 3)
	codes := make([]string, 3)
	go getCore(e, c, 0)
	go getApp(e, c, 1)
	go getCode(e, c, 2)
	for i := 0; i < 3; i++ {
		part := <-c
		codes[part.index] = part.code
	}

	jsCode := fmt.Sprintf(`
		const global = globalThis;
		let __exec__;
		%s
		%s
		const lx = global.lx;
		(function(){
		class Executor {
			constructor() {
				this.logs = {};
				this.errors = {};
				this.dumps = [];
			}
			log(msg, category) { if (!(category in this.logs)) this.logs[category]=[]; if(!lx.isString(msg))msg=JSON.stringify(msg); this.logs[category].push(msg) }
			error(msg, category) { if (!(category in this.errors)) this.errors[category]=[]; this.errors[category].push(msg) }
			dump(msg) { this.dumps.push(msg) }
			run() {
				%s
			}
		}
		__exec__ = new Executor();
		let res;
		try {
			res = {
				log: __exec__.logs,
				errors: __exec__.errors,
				dumps: __exec__.dumps,
				result: __exec__.run(),
				fatal: ''
			};
		} catch (e) {
		    let msg = ['* message:', e.message, '* stack:', e.stack, '* string:', e.toString()];
			msg = msg.join('\n');
			res = {
				log: __exec__.logs,
				errors: __exec__.errors,
				dumps: __exec__.dumps,
				result: null,
				fatal: msg
			};
		}
		return JSON.stringify(res);
		})()
    `, codes[0], codes[1], codes[2])

	result, err := vm.RunString(jsCode)
	if err != nil {
		return nil, err
	}

	jsonStr := result.String()
	var data map[string]any
	err = json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return nil, err
	}

	dumps := make([]string, 0, len(data["dumps"].([]any)))
	for _, v := range data["dumps"].([]any) {
		if s, ok := v.(string); ok {
			dumps = append(dumps, s)
		}
	}

	res := &execResult{
		log:    toStringSliceMap(data["log"].(map[string]any)),
		errors: toStringSliceMap(data["errors"].(map[string]any)),
		dumps:  dumps,
		result: data["result"],
		fatal:  data["fatal"].(string),
	}

	//TODO fmt
	if len(res.Errors()) > 0 || res.Fatal() != "" {
		destPath := "/home/lx/webprj/goprg/lxgo-mchat/frontend/plugins/test_plugin/temp.js"
		_ = os.WriteFile(destPath, []byte(jsCode), 0644)
	}

	return res, nil
}

func toStringSliceMap(input map[string]any) map[string][]string {
	result := make(map[string][]string, len(input))
	for key, val := range input {
		rawSlice, ok := val.([]any)
		if !ok {
			continue
		}
		strSlice := make([]string, 0, len(rawSlice))
		for _, item := range rawSlice {
			if s, ok := item.(string); ok {
				strSlice = append(strSlice, s)
			}
		}
		result[key] = strSlice
	}
	return result
}

func getCore(e *executor, c chan codePart, index int) {
	pp := e.pp
	destPath := pp.App().Pathfinder().GetAbsPath(pp.Config().CorePath)
	re := regexp.MustCompile(`\.js$`)
	destPath = re.ReplaceAllString(destPath, "-server.js")

	data, err := os.ReadFile(destPath)
	if err != nil {
		pp.LogError("can not read server core js-code from file '%s': %s", destPath, err)
		c <- codePart{index: index, code: ""}
		return
	}

	c <- codePart{index: index, code: string(data)}
}

func getApp(e *executor, c chan codePart, index int) {

	_ = e

	c <- codePart{index: index, code: ""}
}

func getCode(e *executor, c chan codePart, index int) {
	code := e.code

	//TODO code validation
	c <- codePart{index: index, code: code}
}
