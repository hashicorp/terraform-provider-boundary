package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"html/template"
	"io"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	shellquote "github.com/kballard/go-shellquote"
	"github.com/sanity-io/litter"
)

var (
	boundaryVersion = "v0.6.1"
	swaggerFile     = fmt.Sprintf("https://raw.githubusercontent.com/hashicorp/boundary/%s/internal/gen/controller.swagger.json", boundaryVersion)
)

func main() {
	if err := Main(); err != nil {
		log.Fatal(err)
	}
}

func Main() error {
	var name, path string
	flag.StringVar(&name, "name", "", "")
	flag.StringVar(&path, "path", "", "")
	flag.Parse()
	if name == "" {
		return fmt.Errorf("the -name flag must be set")
	}
	if path == "" {
		return fmt.Errorf("the -path flag must be set")
	}

	resource, err := NewResourceFromSwagger(swaggerFile, name, "/v1/", path)
	if err != nil {
		return err
	}

	source, err := resource.Render()
	if err != nil {
		return err
	}
	return write(source)
}

type Schema map[string]*schema.Schema

func (s Schema) String() string {
	litter.Config.HideZeroValues = true
	litter.Config.HidePrivateFields = true
	litter.Config.DumpFunc = func(v reflect.Value, w io.Writer) bool {
		switch v.Interface() {
		case schema.TypeString:
			w.Write([]byte("schema.TypeString"))
			return true
		case schema.TypeList:
			w.Write([]byte("schema.TypeList"))
			return true
		case schema.TypeInt:
			w.Write([]byte("schema.TypeInt"))
			return true
		case schema.TypeBool:
			w.Write([]byte("schema.TypeBool"))
			return true
		}
		return false
	}

	res := litter.Sdump(map[string]*schema.Schema(s))
	res = strings.ReplaceAll(res, `": &schema.Schema{`, `": {`)
	return strings.ReplaceAll(res, "schema.ValueType", "")
}

type Resource struct {
	name        string
	path        string
	description string
	schema      Schema
}

func NewResourceFromSwagger(filename, resourceName, root, path string) (*Resource, error) {
	document, err := loads.JSONSpec(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load spec: %w", err)
	}

	swagger := document.Spec()
	// ExpandSpec is not careful to merge the attributes so we loose some
	// descriptions here
	if err = spec.ExpandSpec(swagger, nil); err != nil {
		return nil, fmt.Errorf("failed to expand spec")
	}

	sch := Schema{}

	op := swagger.Paths.Paths[root+path].Get
	for _, param := range op.Parameters {
		s, err := getSchemaFromParameter(param)
		if err != nil {
			return nil, err
		}
		if s == nil {
			continue
		}
		sch[param.Name] = s
	}
	for name, prop := range op.Responses.StatusCodeResponses[200].Schema.Properties {
		s, err := getSchemaFromProperty(name, prop)
		if err != nil {
			return nil, err
		}
		if s == nil {
			continue
		}
		sch[name] = s
	}
	return &Resource{
		name:        resourceName,
		path:        path,
		description: op.Summary,
		schema:      sch,
	}, nil
}

func getSchemaFromParameter(p spec.Parameter) (*schema.Schema, error) {
	s := &schema.Schema{
		Description: p.Description,
		Optional:    !p.Required,
		Required:    p.Required,
	}
	switch ty := p.Type; ty {
	case "string":
		s.Type = schema.TypeString
	case "boolean":
		s.Type = schema.TypeBool
	default:
		return nil, fmt.Errorf("unknwon type %q for %s", ty, p.Name)
	}

	return s, nil
}

func getSchemaFromProperty(name string, p spec.Schema) (*schema.Schema, error) {
	s := &schema.Schema{
		Description: p.Description,
		Computed:    true,
	}
	if len(p.Type) != 1 {
		panic("unexpected")
	}
	switch ty := p.Type[0]; ty {
	case "boolean":
		s.Type = schema.TypeBool
	case "integer":
		s.Type = schema.TypeInt
	case "string":
		s.Type = schema.TypeString
	case "object":
		if len(p.Properties) == 0 {
			return nil, nil
		}
		s.Type = schema.TypeList
		r := &schema.Resource{
			Schema: map[string]*schema.Schema{},
		}
		for n, p := range p.Properties {
			se, err := getSchemaFromProperty(name+"."+n, p)
			if err != nil {
				return nil, err
			}
			if se == nil {
				continue
			}
			r.Schema[n] = se
		}
		s.Elem = r
	case "array":
		s.Type = schema.TypeList
		if len(p.Items.Schema.Properties) != 0 {
			r := &schema.Resource{
				Schema: map[string]*schema.Schema{},
			}
			for n, p := range p.Items.Schema.Properties {
				se, err := getSchemaFromProperty(name+"."+n, p)
				if err != nil {
					return nil, err
				}
				if se == nil {
					continue
				}
				r.Schema[n] = se
			}
			s.Elem = r
		} else {
			se, err := getSchemaFromProperty(name, *p.Items.Schema)
			if err != nil {
				return nil, err
			}
			se.Optional = false
			se.Computed = false
			s.Elem = se
		}
	default:
		return nil, fmt.Errorf("unknwon type %q for %s", ty, name)
	}

	return s, nil
}

func (r *Resource) Render() (string, error) {
	attrs := map[string]schema.ValueType{}
	for name, schema := range r.schema {
		if schema.Optional {
			attrs[name] = schema.Type
		}
	}
	attrs_ := []string{}
	for a := range attrs {
		attrs_ = append(attrs_, a)
	}
	sort.Strings(attrs_)

	imports := map[string]struct{}{
		"context": {},
		"net/url": {},
	}

	var queryParams []string
	for _, attr := range attrs_ {
		switch ty := attrs[attr]; ty {
		case schema.TypeBool:
			imports["strconv"] = struct{}{}
			queryParams = append(queryParams, fmt.Sprintf(`%s := d.Get(%q).(bool)`, attr, attr))
			queryParams = append(queryParams, fmt.Sprintf(`if %s {`, attr))
			queryParams = append(queryParams, fmt.Sprintf(`q.Add(%q, strconv.FormatBool(%s))`, attr, attr))
			queryParams = append(queryParams, `}`)
		case schema.TypeString:
			queryParams = append(queryParams, fmt.Sprintf(`q.Add(%q, d.Get(%q).(string))`, attr, attr))
		default:
			panic(fmt.Sprintf("unknown type %q for %q", ty, attr))
		}

	}

	imports_ := []string{}
	for i := range imports {
		imports_ = append(imports_, i)
	}
	sort.Strings(imports_)

	t := template.Must(template.New("datasource").Parse(`
// Code generated by scripts/generate_datasource.go. DO NOT EDIT.
//go:{{.GenerateLine}}

// This file was generated based on Boundary {{.Version}}

package provider

import (
	{{range .Imports}}"{{.}}"
	{{end}}

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var dataSource{{.Name}}Schema = {{.Schema}}

func dataSource{{.Name}}() *schema.Resource{
	return &schema.Resource{
		Description: {{.Description}},
		Schema: dataSource{{.Name}}Schema,
		ReadContext: dataSource{{.Name}}Read,
	}
}

func dataSource{{.Name}}Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*metaData).client

	req, err := client.NewRequest(ctx, "GET", "{{.Path}}", nil)
	if err != nil {
		return diag.FromErr(err)
	}

	q := url.Values{}
	{{.Query}}
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		diag.FromErr(err)
	}
	apiError, err := resp.Decode(nil)
	if err != nil {
		return diag.FromErr(err)
	}
	if apiError != nil {
		return apiErr(apiError)
	}
	err = set(dataSource{{.Name}}Schema, d, resp.Map)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("boundary-{{.Path}}")

	return nil
}
`))

	w := bytes.NewBuffer(nil)
	data := map[string]interface{}{
		"GenerateLine": fmt.Sprintf("generate go run ../../scripts/generate_datasource.go %s", shellquote.Join("-name", r.name, "-path", r.path)),
		"Version":      boundaryVersion,
		"Imports":      imports_,
		"Name":         r.name,
		"Schema":       template.HTML(r.schema.String()),
		"Description":  template.HTML(fmt.Sprintf("%q", r.description)),
		"Path":         template.HTML(r.path),
		"Query":        template.HTML(strings.Join(queryParams, "\n")),
	}
	if err := t.Execute(w, data); err != nil {
		return "", err
	}

	source, err := format.Source(w.Bytes())
	if err != nil {
		log.Fatalf("the generated go code is incorrect: %s", err)
	}

	return string(source), nil
}

func write(source string) error {
	if filename := os.Getenv("GOFILE"); filename != "" {
		f, err := os.Create(os.Getenv("GOFILE"))
		if err != nil {
			return err
		}
		defer f.Close()
		f.Write([]byte(source))
	} else {
		fmt.Print(source)
	}

	return nil
}
