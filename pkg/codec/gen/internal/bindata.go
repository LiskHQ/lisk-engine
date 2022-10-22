// Code generated for package internal by go-bindata DO NOT EDIT. (@generated)
// sources:
// templates/codec.tmpl
package internal

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
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

var _templatesCodecTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x9c\x94\x41\x4f\x1b\x31\x10\x85\xcf\xeb\x5f\x31\x45\x3d\xac\x11\x78\xef\x45\x5c\xda\x04\x81\x44\x83\x0a\x54\x3d\x54\x3d\x38\xde\x61\x6b\x25\xeb\x5d\x8d\xbd\x42\x91\xe5\xff\x5e\xd9\xde\x84\x2d\x69\x52\xc4\x2d\x9a\x79\xf3\xe6\x7b\x13\x6b\xab\x0a\xbe\x74\x35\x42\x83\x06\x49\x3a\xac\x61\xb9\x81\x46\xbb\xdf\xc3\x52\xa8\xae\xad\x6e\xb5\x5d\x5d\x7f\xab\xd6\xda\xae\xce\xd1\x34\xda\x60\xd5\xaf\x9a\x4a\x75\x35\xaa\xaa\x41\x73\x01\xb3\x3b\x58\xdc\x3d\xc2\x7c\x76\xf3\x28\x18\xeb\xa5\x5a\xc9\x06\xc1\x7b\x10\x0b\xd9\x22\x84\xc0\x98\x6e\xfb\x8e\x1c\x94\xac\xf0\x1e\x48\x9a\x06\xe1\xa3\x6e\x7b\xf8\x74\x09\xe2\x26\xf5\x2c\x9c\x87\xc0\x8a\x13\xef\x63\x23\x84\x93\x24\x45\x53\xc7\x79\xce\xd8\x6e\x4e\x3c\x38\x1a\x94\xb3\xb1\xfe\x34\x18\x05\x25\xc2\xe9\x64\x19\x87\xb9\x89\x70\x25\x87\xf2\xe7\xaf\xe5\xc6\xe1\x19\x20\x51\x47\xdc\xb3\xe2\x99\xb4\x43\x8a\x6b\x13\xbf\x58\xe0\xf3\x8f\x54\x2a\xf9\x04\x4d\x5c\x69\x5c\xd7\x23\x51\x2c\x8b\x6c\x79\xdb\x35\x5a\xc5\xbd\x5b\xb4\x24\x20\x74\x03\x19\xc8\xd6\xe2\x1e\xed\xb0\x76\x25\x3f\x03\xa3\xd7\x2c\xb0\x03\x8c\x5f\x07\xeb\x76\x9c\x19\x13\x3c\x83\x02\x53\xad\x4e\xc8\x11\x13\xc5\x56\xc5\x0a\xd0\x4f\xa9\xfc\xe1\x32\x7a\x83\x67\x05\x14\xbd\x34\x5a\x95\x48\x14\xfb\x2f\x30\xa3\xcd\xe1\xfd\x33\x4c\xae\xb5\x74\x72\x5c\xcf\xf3\x95\xa2\x2d\xa1\xac\x5f\x5d\xe9\x3e\x95\x92\x9e\xbf\x6c\x11\xd9\xe6\x8a\xba\x76\x14\xe4\x51\x7e\x3c\xf8\xbf\x96\x7b\x56\x8c\xf1\x52\xea\x89\x84\x5f\xbc\x4a\xfd\x57\xe8\xf0\xbf\x8c\x0f\x8e\xb4\x72\xef\x4d\xba\xcf\x94\xfd\xf6\x23\xef\x51\x6e\x8f\x44\x14\x29\xa3\x53\x56\x8a\x6b\x69\xbf\x9b\xf8\xfb\xf3\xc6\xa1\x2d\xf9\x54\x9d\x31\xe6\x44\x13\x45\x1a\x1f\xfb\x47\x1f\xd5\x81\x7f\x03\x4e\xb3\x6b\x2e\x4e\xd2\x1f\x79\xf0\xd9\xea\xe8\x83\x7f\x03\xca\x81\x5b\xbd\x1f\x28\x1b\xbe\x01\x6b\xf7\xed\xf8\x13\x00\x00\xff\xff\xca\xee\x02\x4d\xdf\x04\x00\x00")

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

	info := bindataFileInfo{name: "templates/codec.tmpl", size: 1247, mode: os.FileMode(420), modTime: time.Unix(1, 0)}
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
			return nil, fmt.Errorf("asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("asset %s not found", name)
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
			return nil, fmt.Errorf("assetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("assetInfo %s not found", name)
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
				return nil, fmt.Errorf("asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("asset %s not found", name)
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
	err = os.WriteFile(_filePath(dir, name), data, info.Mode())
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
