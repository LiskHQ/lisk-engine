package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	fileName := os.Getenv("GOFILE")
	fullPath := path.Join(cwd, fileName)
	info, err := getPackageInfo(fileName, fullPath)
	if err != nil {
		return err
	}
	if err := info.Validate(); err != nil {
		return err
	}
	info.CleanImports()
	info.Sort()
	baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	byteWriter := bytes.NewBuffer([]byte{})
	if err := generateTemplate(info, byteWriter); err != nil {
		return err
	}
	outputFilePath := path.Join(cwd, fmt.Sprintf("%s_codec.go", baseName))
	if strings.HasSuffix(baseName, "_test") {
		baseWithoutTest := strings.TrimSuffix(baseName, "_test")
		outputFilePath = path.Join(cwd, fmt.Sprintf("%s_codec_test.go", baseWithoutTest))
	}
	writer, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer writer.Close()
	formattedSorce, err := format.Source(byteWriter.Bytes())
	if err != nil {
		return err
	}
	if _, err := writer.Write(formattedSorce); err != nil {
		return err
	}
	return nil
}
