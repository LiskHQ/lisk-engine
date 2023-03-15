package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/structtag"
)

var forbiddenType = []string{
	"uint8",
	"int8",
	"uint16",
	"int16",
	"map",
	"[][][]",
}

type packageInfo struct {
	Name     string
	FileName string
	Imports  []string
	Structs  []*structInfo
}

func (i *packageInfo) Validate() error {
	for _, s := range i.Structs {
		if err := s.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (i *packageInfo) Sort() {
	for _, s := range i.Structs {
		s.Sort()
	}
}

func (i *packageInfo) CleanImports() {
	pkg := map[string]bool{}
	for _, imp := range i.Imports {
		index := strings.LastIndex(imp, "/")
		pkg[imp[index+1:]] = false
	}
	for _, s := range i.Structs {
		for _, f := range s.Fields {
			for name := range pkg {
				if strings.Contains(f.Type, name) {
					pkg[name] = true
				}
			}
		}
	}
	newImports := []string{"github.com/LiskHQ/lisk-engine/pkg/codec"}
	for _, imp := range i.Imports {
		index := strings.LastIndex(imp, "/")
		name := imp[index+1:]
		exist := pkg[name]
		if exist {
			newImports = append(newImports, imp)
		}
	}
	i.Imports = newImports
}

type structInfo struct {
	Name   string
	Fields []*fieldInfo
}

func (i *structInfo) Validate() error {
	fieldNumberMap := map[int]bool{}
	for _, f := range i.Fields {
		_, exist := fieldNumberMap[f.FieldNumber]
		if exist {
			return fmt.Errorf("fieldNumber %d is duplicated on field %s", f.FieldNumber, f.Name)
		}
		fieldNumberMap[f.FieldNumber] = true
		if err := f.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (i *structInfo) Sort() {
	sort.Slice(i.Fields, func(j, k int) bool {
		return i.Fields[j].FieldNumber < i.Fields[k].FieldNumber
	})
}

type fieldInfo struct {
	Name        string
	Type        string
	FieldNumber int
}

func (i *fieldInfo) Validate() error {
	if i.FieldNumber < 1 {
		return fmt.Errorf("fieldNumber must be greater than 1 but got %d", i.FieldNumber)
	}
	for _, keyword := range forbiddenType {
		if strings.Contains(i.Type, keyword) {
			return fmt.Errorf("field contain forbidden keyword %s in %s", keyword, i.Type)
		}
	}
	return nil
}
func (i *fieldInfo) ShowLogic() string {
	return "being called"
}

func (i *fieldInfo) EncodeLogic() string {
	var fn string
	switch i.Type {
	case "bool":
		fn = fmt.Sprintf("writer.WriteBool(%d, e.%s)", i.FieldNumber, i.Name)
	case "[]bool":
		fn = fmt.Sprintf("writer.WriteBools(%d, e.%s)", i.FieldNumber, i.Name)
	case "uint32":
		fn = fmt.Sprintf("writer.WriteUInt32(%d, e.%s)", i.FieldNumber, i.Name)
	case "[]uint32":
		fn = fmt.Sprintf("writer.WriteUInt32s(%d, e.%s)", i.FieldNumber, i.Name)
	case "uint64":
		fn = fmt.Sprintf("writer.WriteUInt(%d, e.%s)", i.FieldNumber, i.Name)
	case "[]uint64":
		fn = fmt.Sprintf("writer.WriteUInts(%d, e.%s)", i.FieldNumber, i.Name)
	case "int32":
		fn = fmt.Sprintf("writer.WriteInt32(%d, e.%s)", i.FieldNumber, i.Name)
	case "[]int32":
		fn = fmt.Sprintf("writer.WriteInt32s(%d, e.%s)", i.FieldNumber, i.Name)
	case "int64":
		fn = fmt.Sprintf("writer.WriteInt(%d, e.%s)", i.FieldNumber, i.Name)
	case "[]int64":
		fn = fmt.Sprintf("writer.WriteInts(%d, e.%s)", i.FieldNumber, i.Name)
	case "string":
		fn = fmt.Sprintf("writer.WriteString(%d, e.%s)", i.FieldNumber, i.Name)
	case "[]string":
		fn = fmt.Sprintf("writer.WriteStrings(%d, e.%s)", i.FieldNumber, i.Name)
	case "[]byte", "codec.Hex", "codec.Lisk32":
		fn = fmt.Sprintf("writer.WriteBytes(%d, e.%s)", i.FieldNumber, i.Name)
	case "[][]byte":
		fn = fmt.Sprintf("writer.WriteBytesArray(%d, e.%s)", i.FieldNumber, i.Name)
	case "[]codec.Lisk32":
		fn = fmt.Sprintf("writer.WriteBytesArray(%d, codec.Lisk32ArrayToBytesArray(e.%s))", i.FieldNumber, i.Name)
	case "[]codec.Hex":
		fn = fmt.Sprintf("writer.WriteBytesArray(%d, codec.HexArrayToBytesArray(e.%s))", i.FieldNumber, i.Name)
	default:
		if strings.Contains(i.Type, "[]") {
			fn = fmt.Sprintf("writer.WriteEncodable(%d, val)", i.FieldNumber)
			loopStart := fmt.Sprintf("{ for _, val := range e.%s {", i.Name)
			loop := fmt.Sprintf("if val != nil { %s }", fn)
			loopEnd := "} }"
			return loopStart + loop + loopEnd
		}
		fn = fmt.Sprintf("writer.WriteEncodable(%d, e.%s)", i.FieldNumber, i.Name)
		result := fmt.Sprintf("if e.%s != nil { %s }", i.Name, fn)
		return result
	}
	return fn
}

func defaultDecodeReturn(fn, name string) string {
	return fmt.Sprintf(`{
		val, err := %s
		if err != nil {
			return err
		}
		e.%s = val
		}`, fn, name)
}

func (i *fieldInfo) DecodeLogic() string {
	return i.decodeLogic("false")
}

func (i *fieldInfo) DecodeStrictLogic() string {
	return i.decodeLogic("true")
}

func (i *fieldInfo) decodeLogic(strict string) string {
	var fn string
	switch i.Type {
	case "bool":
		fn = fmt.Sprintf("reader.ReadBool(%d, %s)", i.FieldNumber, strict)
	case "[]bool":
		fn = fmt.Sprintf("reader.ReadBools(%d)", i.FieldNumber)
	case "uint32":
		fn = fmt.Sprintf("reader.ReadUInt32(%d, %s)", i.FieldNumber, strict)
	case "[]uint32":
		fn = fmt.Sprintf("reader.ReadUInt32s(%d)", i.FieldNumber)
	case "uint64":
		fn = fmt.Sprintf("reader.ReadUInt(%d, %s)", i.FieldNumber, strict)
	case "[]uint64":
		fn = fmt.Sprintf("reader.ReadUInts(%d)", i.FieldNumber)
	case "int32":
		fn = fmt.Sprintf("reader.ReadInt32(%d, %s)", i.FieldNumber, strict)
	case "[]int32":
		fn = fmt.Sprintf("reader.ReadInt32s(%d)", i.FieldNumber)
	case "int64":
		fn = fmt.Sprintf("reader.ReadInt(%d, %s)", i.FieldNumber, strict)
	case "[]int64":
		fn = fmt.Sprintf("reader.ReadInts(%d)", i.FieldNumber)
	case "string":
		fn = fmt.Sprintf("reader.ReadString(%d, %s)", i.FieldNumber, strict)
	case "[]string":
		fn = fmt.Sprintf("reader.ReadStrings(%d)", i.FieldNumber)
	case "[]byte", "codec.Hex", "codec.Lisk32":
		fn = fmt.Sprintf("reader.ReadBytes(%d, %s)", i.FieldNumber, strict)
	case "[][]byte":
		fn = fmt.Sprintf("reader.ReadBytesArray(%d)", i.FieldNumber)
	case "[]codec.Hex":
		fn = fmt.Sprintf("reader.ReadBytesArray(%d)", i.FieldNumber)
		return fmt.Sprintf(`{
		val, err := %s
		if err != nil {
			return err
		}
		e.%s = codec.BytesArrayToHexArray(val)
		}`, fn, i.Name)
	case "[]codec.Lisk32":
		fn = fmt.Sprintf("reader.ReadBytesArray(%d)", i.FieldNumber)
		return fmt.Sprintf(`{
		val, err := %s
		if err != nil {
			return err
		}
		e.%s = codec.BytesArrayToLisk32Array(val)
		}`, fn, i.Name)
	default:
		if strings.Contains(i.Type, "[]") {
			newableType := strings.ReplaceAll(i.Type, "[]*", "")
			pointerType := strings.ReplaceAll(i.Type, "[]", "")
			fn = fmt.Sprintf("reader.ReadDecodables(%d, func () codec.DecodableReader { return new(%s) })", i.FieldNumber, newableType)
			return fmt.Sprintf(`{
		vals, err := %s
		if err != nil {
			return err
		}
		r := make(%s, len(vals))
		for i, v := range vals {
			r[i] = v.(%s)
		}
		e.%s = r
		}`, fn, i.Type, pointerType, i.Name)
		}
		newableType := strings.ReplaceAll(i.Type, "*", "")
		fn = fmt.Sprintf("reader.ReadDecodable(%d, func () codec.DecodableReader { return new(%s) }, %s)", i.FieldNumber, newableType, strict)
		return fmt.Sprintf(`{
		val, err := %s
		if err != nil {
			return err
		}
		e.%s = val.(%s)
		}`, fn, i.Name, i.Type)
	}
	return defaultDecodeReturn(fn, i.Name)
}

func getPackageInfo(fileName, filePath string) (*packageInfo, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, fileName, src, parser.AllErrors)
	if err != nil {
		return nil, err
	}
	info := &packageInfo{
		Name:     node.Name.Name,
		FileName: fileName,
		Imports:  []string{},
		Structs:  []*structInfo{},
	}
	ast.Inspect(node, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.TypeSpec:
			e, ok := t.Type.(*ast.StructType)
			if !ok {
				return true
			}
			structInfo := &structInfo{
				Name:   t.Name.Name,
				Fields: []*fieldInfo{},
			}
			for _, f := range e.Fields.List {
				if f.Tag == nil {
					continue
				}
				// calculate tag, if err, ignore the field
				fieldNumber, err := getFieldNumber(f.Tag.Value)
				if err != nil {
					continue
				}
				fieldInfo := &fieldInfo{
					Name:        f.Names[0].Name,
					Type:        string(src[f.Type.Pos()-1 : f.Type.End()-1]),
					FieldNumber: fieldNumber,
				}
				structInfo.Fields = append(structInfo.Fields, fieldInfo)
			}
			if len(structInfo.Fields) > 0 {
				info.Structs = append(info.Structs, structInfo)
			}
		case *ast.ImportSpec:
			info.Imports = append(info.Imports, strings.ReplaceAll(t.Path.Value, "\"", ""))
		}
		return true
	})
	return info, nil
}

func getFieldNumber(val string) (int, error) {
	tag, err := structtag.Parse(strings.ReplaceAll(val, "`", ""))
	if err != nil {
		return 0, err
	}
	res, err := tag.Get("fieldNumber")
	if err != nil {
		return 0, err
	}
	fieldNumber, err := strconv.Atoi(res.Name)
	if err != nil {
		return 0, err
	}
	return fieldNumber, nil
}
