// Package generator produces Go source files with mapper registration code.
package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"text/template"

	"github.com/KARTIKrocks/gomapper/internal/matcher"
)

// Mode controls what kind of output is generated.
type Mode string

const (
	ModeRegister Mode = "register"
	ModeFunc     Mode = "func"
	ModeBoth     Mode = "both"
)

// PairData holds the template data for a single struct pair.
type PairData struct {
	SrcType              string
	DstType              string
	Mappings             []matcher.FieldMapping
	Unmapped             []matcher.UnmappedField
	NestedDstAssignments []matcher.NestedDstAssignment
}

// Data holds all template data for code generation.
type Data struct {
	PkgName string
	Pairs   []PairData
	NilSafe bool
}

// Parsed templates — allocated once.
var templates = map[Mode]*template.Template{
	ModeRegister: template.Must(template.New("register").Parse(registerTemplate)),
	ModeFunc:     template.Must(template.New("func").Parse(funcTemplate)),
	ModeBoth:     template.Must(template.New("both").Parse(bothTemplate)),
}

// Generate produces formatted Go source code for the given data and mode.
func Generate(data Data, mode Mode) ([]byte, error) {
	tmpl, ok := templates[mode]
	if !ok {
		return nil, fmt.Errorf("unknown mode %q", mode)
	}

	// Fill in SliceDstFull from SliceDst when not explicitly set.
	for i := range data.Pairs {
		for j := range data.Pairs[i].Mappings {
			m := &data.Pairs[i].Mappings[j]
			if m.IsSliceMap && m.SliceDstFull == "" {
				m.SliceDstFull = m.SliceDst
			}
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatting generated code: %w\n\nraw output:\n%s", err, buf.String())
	}

	return formatted, nil
}

// WriteFile generates code and writes it to the given path.
func WriteFile(data Data, mode Mode, path string) error {
	src, err := Generate(data, mode)
	if err != nil {
		return err
	}
	return os.WriteFile(path, src, 0644)
}
