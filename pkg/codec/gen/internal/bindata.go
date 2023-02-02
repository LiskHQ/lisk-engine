// Code generated for package internal by go-bindata DO NOT EDIT. (@generated)
// sources:
// templates/codec.tmpl
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

var _templatesCodecTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x94\x41\x4f\xdb\x40\x10\x85\xcf\xb6\xe4\xff\x30\x45\x3d\x78\x11\xac\xef\x45\x5c\xda\x04\x81\x44\x83\x0a\x54\x3d\x54\x3d\x6c\xd6\x83\xbb\x4a\xbc\xb6\xc6\x6b\xa1\xc8\xf2\x7f\xaf\x76\xc7\x8e\x0c\x81\x24\x2a\x37\x6b\xf6\xcd\x9b\x6f\x9e\xbd\xce\x32\xf8\x56\xe5\x08\x05\x5a\x24\xe5\x30\x87\xe5\x06\x0a\xe3\xfe\xb6\x4b\xa9\xab\x32\xbb\x35\xcd\xea\xfa\x47\xb6\x36\xcd\xea\x1c\x6d\x61\x2c\x66\xf5\xaa\xc8\x74\x95\xa3\xce\x0a\xb4\x17\x30\xbb\x83\xc5\xdd\x23\xcc\x67\x37\x8f\x32\x89\x93\xb8\x56\x7a\xa5\x0a\x84\xae\x03\xb9\x50\x25\x42\xdf\xfb\xb2\x29\xeb\x8a\x1c\xa4\x49\x1c\x75\x1d\x90\xb2\x05\xc2\x67\x53\xd6\xf0\xe5\x12\xe4\x4d\x38\x6c\xe0\xdc\x6b\xa3\x93\xae\xf3\x27\x7d\x7f\xc2\x62\xb4\x79\x30\x11\xde\x67\xdb\x2c\x1f\x1c\xb5\xda\x35\xe1\xe8\xa9\xb5\x1a\x52\x84\xd3\xc9\x54\x01\x73\xeb\x39\x53\x01\xe9\xef\x3f\xcb\x8d\xc3\x33\x40\xa2\x8a\x44\x97\xc4\xd1\x33\x19\x87\xe4\xa7\x87\x5d\xe4\x02\x9f\x7f\x85\x52\x2a\xa6\x88\xf2\xca\xe0\x3a\x1f\xc9\x7c\x5d\xb2\xeb\x6d\x55\x18\x1d\x66\x8f\x88\x2c\x21\x74\x2d\x59\x60\x7b\x79\x8f\x4d\xbb\x76\xa9\x38\x03\x6b\xd6\x49\x1c\x92\x78\x9b\xf5\x7b\xdb\xb8\x2d\x2f\xe3\x42\x97\xc4\x10\x61\x28\xe6\x81\xdd\xe3\xa2\x1c\x65\x49\x1c\x81\x79\x0a\xf5\x4f\x97\x7e\x80\x6f\x88\x20\xaa\x95\x35\x3a\x45\xa2\xa0\x98\x40\x0d\x56\x7b\x39\x66\x18\xcc\x73\xe5\xd4\x80\x21\x38\xb5\x60\x4e\xa8\xf2\x57\xa9\xdd\x87\x52\x68\x10\x93\x51\x92\x8d\xae\xa8\x2a\x07\x05\xf7\x8a\x83\x29\xbc\x45\xe0\x67\x0f\xab\x86\x08\x26\x1a\x71\xf1\x3a\x81\x97\x01\xf4\x47\xac\xfb\xe0\xc8\x68\xf7\xff\x4b\xef\xa2\xb1\xe3\xee\xf6\xbb\xb0\x63\x60\x44\x0c\xeb\xcd\x58\x2c\xaf\x55\xf3\xd3\xfa\xe7\xaf\x1b\x87\x4d\x2a\x5e\x34\x30\xcb\x9c\x68\x22\x19\x1c\x06\xc5\xa1\x4f\xee\x9d\x37\x04\xa7\x6c\xcd\xc5\x69\x10\xfb\x6e\x05\x9b\x1d\xb8\x15\xc7\x11\xbd\x93\xdd\x47\xb8\xd8\xf2\x48\xba\xed\x2f\xe7\x5f\x00\x00\x00\xff\xff\xa4\x2c\x07\xde\x1d\x05\x00\x00")

func templatesCodecTmplBytes() ([]byte, error) {
	return bindataRead(
		_templatesCodecTmpl,
		"templates/codec.tmpl",
	)
}

func templatesCodecTmpl() (*asset, error) {
	bytes, err := templatesCodecTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/codec.tmpl", size: 1309, mode: os.FileMode(511), modTime: time.Unix(1, 0)}
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
	"templates/codec.tmpl": templatesCodecTmpl,
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
		"codec.tmpl": {templatesCodecTmpl, map[string]*bintree{}},
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
