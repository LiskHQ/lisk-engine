package client

import (
	"html/template"
	"io"
	"strings"

	"github.com/LiskHQ/lisk-engine/pkg/client/internal"
)

//go:generate go run github.com/go-bindata/go-bindata/go-bindata -o=internal/bindata.go -pkg=internal -modtime=1 ./templates/...

type moduleTemplateData struct {
	ID   uint32
	Name string
}

type commandTemplateData struct {
	ModuleName string
	ID         uint32
	Name       string
}

type apiTemplateData struct {
	Name string
}

type endpointTemplateData struct {
	Name string
}

type pluginTemplateData struct {
	Name string
}

type mainTemplateData struct {
}

type appTemplateData struct {
}

func toCamel(val string) string {
	return strings.ToUpper(val[:1]) + val[1:]
}

var funcMap = template.FuncMap{
	"ToUpper": strings.ToUpper,
	"ToLower": strings.ToLower,
	"ToTitle": strings.ToTitle,
	"ToCamel": toCamel,
}

func generateModule(info *moduleTemplateData, writer io.Writer) error {
	templateFile := internal.MustAsset("templates/module.tmpl")
	temp, err := template.New("module").Funcs(funcMap).Parse(string(templateFile))
	if err != nil {
		return err
	}
	return temp.Execute(writer, info)
}

func generateCommand(info *commandTemplateData, writer io.Writer) error {
	templateFile := internal.MustAsset("templates/command.tmpl")
	temp, err := template.New("command").Funcs(funcMap).Parse(string(templateFile))
	if err != nil {
		return err
	}
	return temp.Execute(writer, info)
}

func generateAPI(info *apiTemplateData, writer io.Writer) error {
	templateFile := internal.MustAsset("templates/api.tmpl")
	temp, err := template.New("api").Funcs(funcMap).Parse(string(templateFile))
	if err != nil {
		return err
	}
	return temp.Execute(writer, info)
}

func generateEndpoint(info *endpointTemplateData, writer io.Writer) error {
	templateFile := internal.MustAsset("templates/endpoint.tmpl")
	temp, err := template.New("endpoint").Funcs(funcMap).Parse(string(templateFile))
	if err != nil {
		return err
	}
	return temp.Execute(writer, info)
}

func generatePlugin(info *pluginTemplateData, writer io.Writer) error {
	templateFile := internal.MustAsset("templates/plugin.tmpl")
	temp, err := template.New("plugin").Funcs(funcMap).Parse(string(templateFile))
	if err != nil {
		return err
	}
	return temp.Execute(writer, info)
}

func generateMain(info *mainTemplateData, writer io.Writer) error {
	templateFile := internal.MustAsset("templates/main.tmpl")
	temp, err := template.New("main").Funcs(funcMap).Parse(string(templateFile))
	if err != nil {
		return err
	}
	return temp.Execute(writer, info)
}

func generateApp(info *appTemplateData, writer io.Writer) error {
	templateFile := internal.MustAsset("templates/app.tmpl")
	temp, err := template.New("app").Funcs(funcMap).Parse(string(templateFile))
	if err != nil {
		return err
	}
	return temp.Execute(writer, info)
}
