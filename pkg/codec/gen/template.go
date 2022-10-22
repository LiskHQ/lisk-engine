package main

import (
	"html/template"
	"io"

	_ "github.com/go-bindata/go-bindata"

	"github.com/LiskHQ/lisk-engine/pkg/codec/gen/internal"
)

//go:generate go run github.com/go-bindata/go-bindata/go-bindata -o=internal/bindata.go -pkg=internal -modtime=1 ./templates/...

func generateTemplate(info *packageInfo, writer io.Writer) error {
	templateFile := internal.MustAsset("templates/codec.tmpl")
	temp, err := template.New("codec").Parse(string(templateFile))
	if err != nil {
		return err
	}
	return temp.Execute(writer, info)
}
