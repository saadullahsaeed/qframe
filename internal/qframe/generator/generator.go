package generator

import (
	"bytes"
	"github.com/tobgu/qframe/internal/template"
	"strings"
)

//go:generate qfgenerate -source=qframe -dst-file=../../../qframe_gen.go

const viewTemplate = `
// {{.Type}}View provides a "view" into an {{.lowerType}} column and can be used for access to individual elements.
type {{.type}}View struct {
	{{.package}}.View
}

// {{.type}}View returns a view into an {{.lowerType}} column identified by name.
//
// colName - Name of the column.
//
// Returns an error if the column is missing or of wrong type.
// Time complexity O(1).
func (qf QFrame) {{.type}}View(colName string) ({{.type}}View, error) {
	namedColumn, ok := qf.columnsByName[colName]
	if !ok {
		return {{.type}}View{}, errors.New("{{.type}}View", "no such column: %s", colName)
	}

	col, ok := namedColumn.Column.({{.package}}.Column)
	if !ok {
		return {{.type}}View{}, errors.New(
			"{{.type}}View",
			"invalid column type, expected %s, was: %s", "{{.lowerType}}", namedColumn.DataType(),
			reflect.TypeOf(namedColumn.Column))
	}

	return  {{.type}}View{View: col.View(qf.index)}, nil
}

`

func spec(typeName, srcPackage string) template.Spec {
	return template.Spec{
		Name:     typeName,
		Template: viewTemplate,
		Values:   map[string]interface{}{"type": typeName, "lowerType": strings.ToLower(typeName), "package": srcPackage}}
}

func view(typeName, srcPackage string) template.Spec {
	return spec(typeName, srcPackage)
}

func GenerateQFrame() (*bytes.Buffer, error) {
	return template.Generate("qframe", []template.Spec{
		view("Int", "icolumn"),
		view("Float", "fcolumn"),
		view("Bool", "bcolumn"),
		view("String", "scolumn"),
		view("Enum", "ecolumn"),
	}, []string{
		"reflect",
		"github.com/tobgu/qframe/errors",
		"github.com/tobgu/qframe/internal/icolumn",
		"github.com/tobgu/qframe/internal/fcolumn",
		"github.com/tobgu/qframe/internal/bcolumn",
		"github.com/tobgu/qframe/internal/scolumn",
		"github.com/tobgu/qframe/internal/ecolumn",
	})
}
