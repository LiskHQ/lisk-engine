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

var _templatesCodecTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x9c\x93\x41\x6b\xdc\x30\x10\x85\xcf\xd6\xaf\x98\x86\x1e\xac\x40\xe4\x7b\x43\x2e\xed\x6e\x48\x20\xdd\xd0\x24\xa5\x87\xd2\x83\x56\x9e\xb8\x62\x6d\xd9\x8c\x64\xc2\x22\xf4\xdf\x8b\xa4\xdd\xc5\xd4\xc9\x26\xec\xcd\x3c\xbd\x79\xef\x1b\x21\x57\x15\x7c\xeb\x6b\x84\x06\x0d\x92\x74\x58\xc3\x7a\x0b\x8d\x76\x7f\xc7\xb5\x50\x7d\x57\xdd\x69\xbb\xb9\xf9\x51\xb5\xda\x6e\x2e\xd0\x34\xda\x60\x35\x6c\x9a\x4a\xf5\x35\xaa\xaa\x41\x73\x09\x8b\x7b\x58\xdd\x3f\xc1\x72\x71\xfb\x24\x18\x1b\xa4\xda\xc8\x06\xc1\x7b\x10\x2b\xd9\x21\x84\xc0\x98\xee\x86\x9e\x1c\x94\xac\xf0\x1e\x48\x9a\x06\xe1\xb3\xee\x06\xf8\x72\x05\xe2\x36\x9d\x59\xb8\x08\x81\x15\x67\xde\xc7\x83\x10\xce\x92\x15\x4d\x1d\xe7\x39\x63\x87\x39\xf1\xe8\x68\x54\xce\x46\xfd\x79\x34\x0a\x4a\x84\xf3\x49\x19\x87\xa5\x89\x70\x25\x87\xf2\xf7\x9f\xf5\xd6\x21\x07\xcf\x8a\x17\xd2\x0e\x29\x16\x26\x72\xb1\xc2\x97\x5f\x49\x2a\xf9\x04\x4a\x5c\x6b\x6c\xeb\x1d\x4b\x94\x45\x0e\xbb\xeb\x1b\xad\x62\xe3\x1e\x2a\x19\x08\xdd\x48\x06\x72\xb4\x78\x40\x3b\xb6\xae\xe4\x2c\xb0\x37\xc0\x16\x98\xc0\x6a\xe9\x24\xec\xd1\x90\xa8\xa7\x08\x48\x28\xeb\xff\x00\x1f\x92\x94\xfc\xfc\xd0\x86\x22\xc7\x5c\x53\xdf\xed\x0c\x79\xf4\x48\xf1\xf7\xd1\xba\xd7\xca\x3d\x2b\xf4\x73\x44\x88\xb5\xfb\xe0\xdc\x77\x99\xe4\x4f\x57\x60\x74\x1b\x7d\xc5\x20\x8d\x56\x25\x12\x71\x56\x84\xf7\x76\x7c\x74\xa4\x95\x3b\x75\xd3\x39\x53\xce\x9b\xaf\x3c\xa3\xdc\x5f\x12\x51\xa4\x8c\x49\xd9\x29\x6e\xa4\xfd\x69\xe2\xf7\xd7\xad\x43\x5b\xf2\xa9\x3b\x63\x2c\x89\x26\x8e\x34\xbe\x3b\x37\xba\x7d\x6f\xe1\x19\x1a\x9c\xe7\xd4\x2c\x4e\xb6\x3f\xf2\xd6\x72\xd4\xd1\xb7\xf6\x01\x94\x37\xee\xea\x74\xa0\x1c\xf8\x01\xac\xc3\x0f\xfb\x2f\x00\x00\xff\xff\x9a\x73\x0e\x7d\x54\x04\x00\x00")

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

	info := bindataFileInfo{name: "templates/codec.tmpl", size: 1108, mode: os.FileMode(420), modTime: time.Unix(1, 0)}
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
