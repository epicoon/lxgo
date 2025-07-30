package cmd

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/epicoon/lxgo/cmd"
	"github.com/epicoon/lxgo/kernel"
)

type ApiDocCommandOptions struct {
	Router kernel.IRouter
	Output string
}

/** @interface cmd.ICommand */
type ApiDocCommand struct {
	*cmd.Command
	router kernel.IRouter
	output string
}

/** @type cmd.FConstructor */
func NewApiDocCommand(opt ...cmd.ICommandOptions) cmd.ICommand {
	options := cmd.GetOptions[ApiDocCommandOptions](opt)
	c := &ApiDocCommand{
		Command: cmd.NewCommand(),
		router:  options.Router,
		output:  options.Output,
	}
	c.RegisterActions(cmd.ActionsList{
		"gen": gen,
	})
	return c
}

type endpoint struct {
	path    string
	method  string
	handler kernel.IHttpResource
	md      string
}

func gen(com cmd.ICommand) error {
	c := com.(*ApiDocCommand)

	counter := 1
	var sb strings.Builder
	sb.WriteString("# API\n")

	endpoints := getEndpoints(c.router.Resources())
	for i := range endpoints {
		generateEndpointMD(&endpoints[i], &sb, counter)
		counter++
	}
	for i := range endpoints {
		sb.WriteString(endpoints[i].md)
	}

	if c.output == "" {
		fmt.Println(sb.String())
	} else {
		if err := os.WriteFile(c.output, []byte(sb.String()), 0644); err != nil {
			fmt.Printf("Can not write file '%s': %v\n", c.output, err)
		}
		fmt.Println("Done")
	}

	return nil
}

func generateEndpointMD(ep *endpoint, commonSB *strings.Builder, counter int) {
	commonSB.WriteString(fmt.Sprintf("* [%s `%s`](#r%d)\n", ep.path, ep.method, counter))

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("<a name=\"r%d\"></a>\n", counter))
	sb.WriteString(fmt.Sprintf("## Path: `%s`\n", ep.path))
	sb.WriteString(fmt.Sprintf("### Method: `%s`\n", ep.method))

	cReqForm := ep.handler.CRequestForm()
	if cReqForm != nil {
		reqForm := cReqForm()
		sb.WriteString("### Request Params\n")
		generateFormMD(reqForm, &sb)
	}

	cRespForm := ep.handler.CResponseForm()
	if cRespForm != nil {
		respForm := cRespForm()
		sb.WriteString("### Response Body\n")
		generateFormMD(respForm, &sb)
	}

	cFailForm := ep.handler.CFailForm()
	if cFailForm != nil {
		failForm := cFailForm()
		sb.WriteString("### Failed Response Body\n")
		generateFormMD(failForm, &sb)
	}

	ep.md = sb.String()
}

func generateFormMD(f kernel.IForm, sb *strings.Builder) {
	sb.WriteString("| field name    | type   | obligate | description |\n")
	sb.WriteString("| ------------- | ------ | -------- | ----------- |\n")

	conf := f.Config()
	formType := reflect.TypeOf(f).Elem()

	for i := 0; i < formType.NumField(); i++ {
		field := formType.Field(i)
		fieldName := field.Tag.Get("json")
		if fieldName == "" {
			continue
		}

		cfg, exists := conf[fieldName]
		if !exists {
			continue
		}

		fieldType := field.Type.String()
		obligate := "no"
		if cfg.Required {
			obligate = "yes"
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", fieldName, fieldType, obligate, cfg.Description))
	}
}

func getEndpoints(resList map[string]kernel.HttpResourcesList) []endpoint {
	var forAll *endpoint
	endpoints := make([]endpoint, 0)
	for path, list := range resList {
		forAll = nil
		for method, handler := range list {
			h := handler()
			if h.CRequestForm() == nil && h.CResponseForm() == nil && h.CFailForm() == nil {
				continue
			}
			ep := endpoint{
				path:    path,
				method:  method,
				handler: h,
			}
			if method == "ALL" {
				forAll = &ep
				continue
			}
			endpoints = append(endpoints, ep)
		}
		if forAll != nil {
			endpoints = append(endpoints, *forAll)
		}
	}
	return endpoints
}
