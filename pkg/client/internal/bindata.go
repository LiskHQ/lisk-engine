// Code generated for package internal by go-bindata DO NOT EDIT. (@generated)
// sources:
// templates/api.tmpl
// templates/app.tmpl
// templates/command.tmpl
// templates/endpoint.tmpl
// templates/main.tmpl
// templates/module.tmpl
// templates/plugin.tmpl
package internal

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _templatesApiTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x5c\x8e\x4d\x4a\xc4\x40\x10\x46\xd7\x29\xa8\x3b\x14\xb3\x4a\x04\xd3\xa0\x6b\x17\x01\x17\x06\x06\x89\xe0\x05\x7a\x62\x4d\x5b\x74\xfa\x87\x4e\x35\x83\xc4\xdc\x5d\x1c\x70\x16\xee\x3e\x3e\x1e\x8f\x97\xed\xec\xad\x63\xda\x36\xea\x5f\x6d\x60\xfa\xa6\xf7\x74\x4c\x17\x2e\xb4\xef\x08\x08\x12\x72\x2a\x4a\x2d\x42\x73\x70\xa2\x9f\xf5\xd4\xcf\x29\x98\xa3\xac\xfe\xe5\xcd\x2c\xb2\xfa\x7b\x8e\x4e\x22\x9b\xec\x9d\x39\x17\x1b\xf8\x92\x8a\x37\xa7\xa5\x72\x2e\x12\xf5\x80\xd0\xfd\x8a\xf4\x2b\x33\x0d\xd3\x48\xab\x96\x3a\x2b\x6d\x08\xcd\x0d\xea\x87\x69\x44\x68\x42\xfa\xa8\x0b\x8f\xcf\x54\x25\xea\xe3\x03\xc2\x35\xe1\x5c\xe3\x4c\xad\xa5\xbb\x61\x1a\x3b\x92\x28\xda\xfe\x03\xbb\xab\xcd\xf6\xb7\xfb\x89\xfe\x26\xc2\xfe\x13\x00\x00\xff\xff\x36\x84\x75\x25\xe4\x00\x00\x00")

func templatesApiTmplBytes() ([]byte, error) {
	return bindataRead(
		_templatesApiTmpl,
		"templates/api.tmpl",
	)
}

func templatesApiTmpl() (*asset, error) {
	bytes, err := templatesApiTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/api.tmpl", size: 228, mode: os.FileMode(511), modTime: time.Unix(1, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesAppTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x54\x4d\x8f\xd3\x30\x10\x3d\x27\x52\xfe\x83\xe9\x01\x39\xab\xc5\xb9\x2f\xea\x61\xf9\x2a\x20\xbe\x25\x4e\x08\x21\xd7\x9d\xa4\x43\x12\xdb\xb2\x27\x44\xab\xaa\xff\x1d\xd9\xce\x76\x53\xc4\x2e\xad\x38\x25\x9e\x99\x37\xf3\x3c\xf3\x3c\x56\xaa\x56\x36\xc0\x7a\x89\xba\xc8\x8b\x1c\x7b\x6b\x1c\x31\x5e\xe4\xd9\x02\xb4\x32\x1b\xd4\x4d\xf5\xd3\x1b\xbd\x08\x16\x34\x15\x9a\x81\xb0\x8b\x27\xe3\xe3\xc7\x4a\xda\x2e\x02\x36\x5b\x34\x48\xdb\x61\x2d\x94\xe9\xab\x77\xe8\xdb\xd7\x9f\xab\x0e\x7d\xfb\x04\x74\x83\x1a\x2a\xdb\x36\xd5\xba\x33\xaa\x55\x5b\x89\x29\xe1\xbf\x01\xaa\x43\xd0\x74\x62\x70\xed\x64\x0f\xa3\x71\x6d\xa5\x8c\xae\xb1\x39\x1b\x66\x1d\x78\x08\xd5\xca\x70\x21\xba\xb1\xc0\x3c\x49\x47\xe0\x98\x27\x37\x28\xda\xed\x83\xa3\x1e\xb4\x62\xdc\xb3\x8b\xc9\x59\xb2\x15\xd0\xb5\xb5\x1d\x2a\x49\x68\x34\x57\xa6\xef\xa5\xde\x3c\x8f\x2c\xd8\x45\x62\x23\x66\x11\xc9\x53\x32\x9e\xee\x37\x77\x5d\x32\x70\xce\xb8\x92\xed\x8a\x3c\x4b\x48\x76\xb5\x64\x8f\xef\x4b\x12\x28\x65\xe3\x26\xc2\x42\xa0\xf1\x62\x05\x34\x6e\x78\x59\xe4\x19\xd6\xd1\xfc\x68\xc9\x34\x76\x31\x63\xe6\x80\x06\xa7\xc3\x39\x42\x8a\x3c\xdb\x1f\x0a\xbd\xc2\x0e\xe6\x89\xbe\x80\xdc\x04\x1b\x0f\x43\x16\x6f\x0d\x6a\x1e\x2a\x2d\xc4\xd4\x60\x11\xa5\x51\x9e\x55\x69\x0a\xbc\x5a\xb2\x00\x16\x5f\x75\x2f\x9d\xdf\xca\x8e\xcf\x29\xa4\xff\xf2\xe9\xd9\x39\x27\x5e\x6f\xb4\x07\x47\x2f\xa0\x96\x43\x47\xfc\xd4\x3c\x45\x9e\x55\x15\xfb\xf8\x0b\xdc\xe8\x90\x80\x4d\x63\x64\xa8\xed\x40\x87\x26\x89\xf7\xe0\x1a\x38\x9e\x71\xd4\x4b\x96\xe4\x73\x6d\x6d\x60\x92\x0e\xe2\x03\x8c\x9f\x6e\xcd\x33\x7d\x4c\xa0\x5b\x1e\x07\x64\x98\xef\x65\xa0\x55\xe4\xf7\x4b\x6d\x05\x1a\x3c\xfa\x67\xe1\x31\x9d\xa1\xb5\x6f\xdf\xd7\x37\x04\x47\xfa\xfa\x6f\xdd\x34\x89\xca\xe9\xc2\x99\x00\x3f\xe2\x26\x78\x58\x3f\x56\x6a\x54\x1c\x9c\x2b\xa7\x62\x11\x13\x1f\xc3\xdd\x1e\x11\xb1\x0b\xbb\x07\x95\x75\x44\x32\x42\xff\xa2\x88\x3f\xab\xdd\x0d\x27\x51\x7d\x19\xd6\x21\x84\xee\xec\x8b\xfc\x77\x00\x00\x00\xff\xff\xfb\x58\x97\x56\x38\x05\x00\x00")

func templatesAppTmplBytes() ([]byte, error) {
	return bindataRead(
		_templatesAppTmpl,
		"templates/app.tmpl",
	)
}

func templatesAppTmpl() (*asset, error) {
	bytes, err := templatesAppTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/app.tmpl", size: 1336, mode: os.FileMode(511), modTime: time.Unix(1, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesCommandTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x94\x91\x41\xcb\xda\x40\x10\x86\xcf\x06\xf2\x1f\x86\x9c\x12\xa1\x59\x68\x6f\xbd\x6a\xa1\x82\x2d\x6d\x91\xde\xd7\x75\x5c\x97\x64\x67\xc3\x64\x16\x15\x9b\xff\x5e\x56\x83\x04\x3e\x3e\x3f\x3d\xcf\x3b\xcf\x3e\xb3\x6f\xa7\x4d\xa3\x2d\xc2\xe5\x02\xf5\x8f\xb0\x8b\x2d\xfe\xd4\x1e\xe1\x1f\x6c\xc2\x3a\x1c\x91\x61\x18\xf2\x2c\xcf\x9c\xef\x02\x0b\x94\x79\x36\x2b\xac\x93\x43\xdc\xd6\x26\x78\xb5\x76\x7d\xf3\xfd\xb7\x6a\x5d\xdf\x7c\x42\xb2\x8e\x50\x75\x8d\x55\x7b\xd6\x1e\x8f\x81\x1b\xb5\x6d\x23\x76\xec\x48\x8a\xe7\x36\x7b\xd1\x82\x5e\x9b\x83\x23\x2c\xf2\xac\x4a\x6f\x2b\x65\xc3\x57\x8b\x84\xac\x05\xc1\x06\xe0\x48\xf0\x31\xca\x84\x1d\x1a\x65\x91\x12\x43\xce\x5d\xba\xb1\xbe\x1f\xb7\xd0\x1e\xdb\x61\xf8\xa5\x59\xfb\x1e\x7a\xe1\x68\x04\x2e\x79\x36\x3c\x4a\x2f\x82\xf7\x9a\x76\x93\xf8\xec\x7e\x60\x3d\x0e\xf3\x6c\xe6\xaf\xff\xb8\x5a\x42\x74\x24\x5f\x3e\x8f\xd0\x7d\x24\x03\xa5\x81\xf9\xfb\xe0\x0a\x56\xcb\xb2\x1a\xd7\xae\x78\x46\x89\x4c\xc9\x65\xb5\x4c\x4d\x3c\x4f\x4a\x83\xb2\x4a\xaa\x8e\xec\x94\x55\x4c\xb7\xae\x1d\x0f\x43\xf1\x0a\xf9\x2f\xb2\xdb\x9f\x4b\x23\x27\x98\x4f\xfb\xaa\x37\xac\xa9\xd7\x46\x5c\xa0\x5b\x66\x11\x48\xf0\x24\xc9\x62\x12\xbb\xcd\xfe\x60\x1f\x5b\x99\x8a\x91\x6b\x5f\xd1\xf8\x76\x42\x13\x05\x1f\x7b\x8c\xa1\xbb\x08\x32\x07\x7e\xfb\xe8\xff\x00\x00\x00\xff\xff\x79\x63\x58\xd9\x06\x03\x00\x00")

func templatesCommandTmplBytes() ([]byte, error) {
	return bindataRead(
		_templatesCommandTmpl,
		"templates/command.tmpl",
	)
}

func templatesCommandTmpl() (*asset, error) {
	bytes, err := templatesCommandTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/command.tmpl", size: 774, mode: os.FileMode(511), modTime: time.Unix(1, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesEndpointTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x8e\xc1\x4a\xc4\x30\x10\x86\xcf\x0d\xe4\x1d\x86\x9e\x5a\xc1\x16\xf4\xec\x4d\x71\x85\x45\x10\xbc\x89\x87\x6c\x3a\xdb\x1d\xda\x4c\x42\x32\x61\x91\xda\x77\x17\x0b\x2d\xea\x41\xf6\x36\xcc\xfc\xff\x37\x5f\x30\x76\x30\x3d\xc2\x34\x41\xf3\x6c\x1c\xc2\x27\xbc\xfa\xbd\x3f\x63\x84\x79\xd6\x4a\x2b\x72\xc1\x47\x81\x4a\xab\xa2\xec\x49\x4e\xf9\xd0\x58\xef\xda\x3d\xa5\x61\xf7\xd2\x8e\x94\x86\x6b\xe4\x9e\x18\xdb\x30\xf4\xed\x31\x1a\x87\x67\x1f\x87\xf6\x30\x66\x0c\x91\x58\xca\xcb\x9a\x31\xd8\x52\xab\xfa\xfb\xa5\x7c\x04\x84\x07\xee\x82\x27\x16\x48\x12\xb3\x15\x98\xb4\x2a\x36\x66\xb3\x5e\xb5\x2a\x9c\xef\xf2\x88\x4f\xf7\x90\x89\xe5\xf6\x46\xab\x45\xfb\x98\xd9\x42\x85\x70\xb5\x26\x6b\x20\x26\xa9\xfe\xa4\xeb\x85\x8b\xcd\xb6\xbe\x83\x75\xfc\x05\x32\x3f\x41\x8f\x28\x55\x0d\x49\x8c\xa0\x33\xf6\x44\x8c\x9b\xcf\xce\x70\x37\x62\x4c\x0b\x36\xa2\xe4\xc8\xe0\x4c\x78\x4b\x12\x89\xfb\xf7\xff\x3a\xd3\xac\xd5\xfc\x15\x00\x00\xff\xff\xa3\x6a\x1c\x6c\x8f\x01\x00\x00")

func templatesEndpointTmplBytes() ([]byte, error) {
	return bindataRead(
		_templatesEndpointTmpl,
		"templates/endpoint.tmpl",
	)
}

func templatesEndpointTmpl() (*asset, error) {
	bytes, err := templatesEndpointTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/endpoint.tmpl", size: 399, mode: os.FileMode(511), modTime: time.Unix(1, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesMainTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x8f\x41\x4f\x33\x21\x10\x86\xcf\x90\xf0\x1f\xe6\xe3\xf0\x65\xd7\x54\x36\xf1\xd8\xa4\x87\xa6\xc6\x6a\xec\x45\x1b\x4f\xc6\x03\xe2\x14\xc9\xb2\x40\x80\xed\xa5\xd9\xff\x6e\xa6\xdb\x78\xe8\xc9\x0b\x84\x79\x5e\x9e\x99\x49\xda\xf4\xda\x22\x0c\xda\x05\xc1\x05\x77\x43\x8a\xb9\x42\x23\x38\x93\x3e\x5a\x49\x77\x2c\x92\x10\x93\xd6\xd5\xef\xf1\x53\x99\x38\x74\x3b\x57\xfa\xc7\x97\xce\xbb\xd2\xdf\x62\xb0\x2e\x60\x97\x7a\xdb\x19\xef\x30\x54\x79\x15\x1e\xf3\x41\x1f\x91\x60\x77\xbc\x93\x82\xb7\xa4\x3b\x8c\xc1\x9c\xdb\x36\x2d\x9c\x04\x67\x3a\x25\x58\xae\xc0\x78\xa7\xd6\x29\x51\x85\xbd\x15\x6d\x71\x09\x92\x9a\xc1\x36\x7a\x1d\x2c\xec\xef\x9f\x61\xb3\x7b\x82\x1a\xa3\x97\x0b\x4a\x6d\xe2\x30\xe8\xf0\x55\x96\xf0\xfe\x71\x43\xdf\x2f\x85\xb3\x82\xcd\x13\xa9\x2d\xd6\x7d\xd5\xb9\x5e\x58\xf3\xbf\xd0\x0b\xf3\x69\x6a\x17\x57\xb9\x2d\x06\x2c\xae\xfc\x21\xb9\x36\x26\x8e\xe1\xd7\x39\xf3\x89\xce\x49\x70\x86\x39\xd3\x42\x3a\x25\xf5\x3a\x86\x26\x16\xb5\xce\xb6\xb4\x82\x33\x77\x00\x82\xff\x56\x10\x9c\x3f\x2f\xcf\x7c\xb4\xea\x41\x57\xed\x1b\xcc\xb9\x9d\x0d\x93\xe0\x3f\x01\x00\x00\xff\xff\x97\x9e\x7c\xe1\x9f\x01\x00\x00")

func templatesMainTmplBytes() ([]byte, error) {
	return bindataRead(
		_templatesMainTmpl,
		"templates/main.tmpl",
	)
}

func templatesMainTmpl() (*asset, error) {
	bytes, err := templatesMainTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/main.tmpl", size: 415, mode: os.FileMode(511), modTime: time.Unix(1, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesModuleTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xbc\x93\x4f\x6f\xd3\x30\x18\xc6\xcf\x8d\x94\xef\x60\xf5\x30\x25\xd5\x48\x25\xb8\xed\xd6\xfd\x61\x44\x8c\x51\xd8\xc4\x05\x71\x70\xbd\x37\xd9\xab\x24\xaf\x23\xdb\x51\x37\x42\xbe\x3b\xb2\x93\x8c\x64\x23\x50\x56\xa0\xa7\xea\xfd\xf3\x7b\xfc\x3c\x76\x4a\x2e\x32\x9e\x02\xab\x6b\x16\x5d\xf2\x02\xd8\x37\x76\x2d\x2f\xe4\x16\x14\x6b\x1a\xdf\xf3\x3d\x2c\x4a\xa9\x0c\x0b\x7c\x6f\x36\x4f\xd1\xdc\x56\x9b\x48\xc8\x62\x79\x81\x3a\x7b\xf3\x61\x99\xa3\xce\x5e\x00\xa5\x48\xb0\x2c\xb3\x74\x99\x28\x5e\xc0\x56\xaa\x6c\xb9\xc9\x2b\x28\x15\x92\x99\xff\xe9\xa6\x90\x94\x60\xba\xe3\x5a\x0a\x04\x8a\x1b\xa9\x76\x9c\x57\xa5\xd8\x71\x52\x1b\x6e\xa0\xe0\xe2\x16\x09\xe6\xbe\x17\xda\x30\xcc\x7d\x09\xec\x9d\xbc\xa9\x72\x60\xda\xa8\x4a\x18\x56\xfb\xde\xec\xc1\x6b\xd4\xf6\x7c\x6f\xc6\x4b\x64\xee\xb7\x58\xad\x63\xdf\x9b\x01\xdd\x94\x12\xc9\xb0\xc5\x59\xf7\xcf\xf7\x5c\xbe\x49\x45\x82\x5d\xc2\xb6\xdd\x0c\x42\xb6\xe8\xf8\x16\xac\xc0\x54\x8a\xd8\x41\x5b\xb2\x15\x0b\x3e\x6a\xc9\x07\xab\x75\x5c\x37\x87\xb6\xd8\xd3\x8f\xd8\x41\x8f\x6f\x3b\xcd\x50\x25\x28\x7a\x78\xc8\xe2\xd3\x20\x64\x15\x92\x79\xf5\x72\xa8\x54\xd7\x51\x7c\xda\x4c\x6e\xd9\x17\x12\x84\xd6\x3a\x52\x3a\xdc\x9b\xd7\xf5\xa3\xe7\xd3\x34\xf3\x29\xca\x6a\x1d\x5b\x9f\xab\x75\x3c\x44\x14\x11\x2f\x71\x6a\xa5\x77\xe5\xc4\x7f\xdc\x4b\xd4\xd7\xc7\x20\x18\x46\xfc\xd4\x3a\xa1\x09\x44\x92\xb2\xcf\x5f\x36\xf7\x06\x42\x06\x4a\x49\xe5\x08\xee\x0c\x11\xda\x81\x22\xb2\x11\x85\xae\xd8\xf3\x1e\x77\x3a\x41\xc2\x7c\xea\xdc\xe7\x60\x4e\x64\x51\x70\xba\x09\xc8\xc6\xd3\x26\x17\xb2\x60\xe4\xa2\x1b\x39\x64\x1b\x29\xf3\x70\xe8\x85\x30\x3f\x64\x09\xcf\x35\x4c\x5e\x24\xa1\x39\xce\xa5\xc8\x02\x61\xee\xd8\xc3\xe7\x10\xb9\xda\x39\xd0\x89\x24\x03\x77\x66\xe8\xf2\xf7\xc7\xbe\x02\x9e\xff\x75\xe8\x27\x50\x98\xdc\xaf\xb4\x06\xa3\x1d\x77\x94\x81\x43\xb7\x23\x7b\xd0\xaf\x15\x27\xcd\x85\x41\x49\x4e\x62\x31\xd2\x18\x74\x1f\x29\x8d\xc6\xda\xde\x47\xd0\x55\x3e\x7a\x58\xa3\xa1\x4b\xd8\x0e\xe7\xde\xbf\x0d\xc2\x5f\x5d\xd1\x39\x10\x68\xd4\x57\x16\xf1\x93\x93\x75\x6d\x17\xc2\x5a\x49\x01\x5a\x23\xa5\xcf\x0a\xe2\x35\x12\xcf\xf1\x2b\xfc\x3f\xc5\x63\x48\xa4\x82\x41\xb8\xfa\xec\x0e\x44\xd5\xc9\x3e\xbd\xe5\xae\xfb\x2c\xad\x55\x62\x40\xed\x2e\xe5\xc6\xf7\xd1\x6b\xbd\x75\x1f\xe8\x50\x6b\xf2\x65\xed\xed\xee\x1f\x88\x7d\x0f\x00\x00\xff\xff\x78\x92\x22\x25\xe6\x07\x00\x00")

func templatesModuleTmplBytes() ([]byte, error) {
	return bindataRead(
		_templatesModuleTmpl,
		"templates/module.tmpl",
	)
}

func templatesModuleTmpl() (*asset, error) {
	bytes, err := templatesModuleTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/module.tmpl", size: 2022, mode: os.FileMode(511), modTime: time.Unix(1, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesPluginTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x94\x91\x31\x4f\xc3\x30\x10\x85\xe7\x44\xca\x7f\x38\x32\x54\x71\x05\xc9\x0e\xea\xc4\x02\x52\x55\x81\x80\x1f\xe0\x1a\xc7\x3d\x92\x9c\xad\x8b\xa3\x0a\x85\xfc\x77\xe4\xba\x0d\x1d\xda\x81\xf5\xde\xf7\x9e\xef\x9d\x9d\x54\x8d\x34\x1a\xc6\xb1\xdc\xc8\x4e\xc3\x0f\xbc\xdb\xb5\xdd\x6b\x9e\xa6\x2c\xcd\x52\xec\x9c\x65\x0f\x45\x96\x26\xb9\x26\x65\x3f\x91\x4c\xf5\xd5\x5b\xca\x83\x9a\xe4\x06\xfd\x6e\xd8\x96\xca\x76\xd5\x1a\xfb\xe6\xe9\xb5\x6a\xb1\x6f\xee\x34\x19\x24\x5d\xb9\xc6\x54\x35\xcb\x4e\xef\x2d\x37\xf9\x3f\xf9\x6a\xdb\x0e\xda\x31\x92\xcf\xb3\x54\x84\xe7\xfc\xb7\xd3\xf0\x68\xa9\x46\x03\xbd\xe7\x41\x79\x18\xb3\x74\x9a\xa5\x97\x76\x30\x48\x67\x52\x32\x47\x94\x51\x3b\xd2\xf5\x40\x0a\x36\x7a\x1f\x87\x85\x80\xe5\xd1\x1a\x3c\xac\xfd\xc0\x04\x8b\x38\x1a\xa7\x73\x4f\xe1\x4e\xa8\x80\x70\xae\x42\x84\xd7\x90\xcc\xb9\x33\xbf\x70\xcb\xfc\x5a\xca\x33\xa1\x2f\x54\x6d\x60\x39\x17\x3f\xee\x1a\x8b\x0a\xd0\xcc\x96\x0f\xf9\x2a\x56\xbf\x5f\xc1\x22\x8a\x61\xb9\x04\xeb\x80\x84\x69\xf8\x97\xf2\x83\x3a\xc9\xfd\x4e\xb6\x21\xb5\x8c\xdc\x2d\x44\xab\x78\x38\xa0\x37\x2b\x20\x6c\x0f\x91\xa7\x9d\x35\x73\x96\x26\xd3\x5f\x09\xc2\xf6\xda\xca\x6f\x5e\xb2\x2f\xc4\x7c\xfb\x0b\x80\x75\xb3\xfe\x1b\x00\x00\xff\xff\x11\xfb\xfa\x13\x61\x02\x00\x00")

func templatesPluginTmplBytes() ([]byte, error) {
	return bindataRead(
		_templatesPluginTmpl,
		"templates/plugin.tmpl",
	)
}

func templatesPluginTmpl() (*asset, error) {
	bytes, err := templatesPluginTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/plugin.tmpl", size: 609, mode: os.FileMode(511), modTime: time.Unix(1, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"templates/api.tmpl":      templatesApiTmpl,
	"templates/app.tmpl":      templatesAppTmpl,
	"templates/command.tmpl":  templatesCommandTmpl,
	"templates/endpoint.tmpl": templatesEndpointTmpl,
	"templates/main.tmpl":     templatesMainTmpl,
	"templates/module.tmpl":   templatesModuleTmpl,
	"templates/plugin.tmpl":   templatesPluginTmpl,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//
//	data/
//	  foo.txt
//	  img/
//	    a.png
//	    b.png
//
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"templates": {nil, map[string]*bintree{
		"api.tmpl":      {templatesApiTmpl, map[string]*bintree{}},
		"app.tmpl":      {templatesAppTmpl, map[string]*bintree{}},
		"command.tmpl":  {templatesCommandTmpl, map[string]*bintree{}},
		"endpoint.tmpl": {templatesEndpointTmpl, map[string]*bintree{}},
		"main.tmpl":     {templatesMainTmpl, map[string]*bintree{}},
		"module.tmpl":   {templatesModuleTmpl, map[string]*bintree{}},
		"plugin.tmpl":   {templatesPluginTmpl, map[string]*bintree{}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
