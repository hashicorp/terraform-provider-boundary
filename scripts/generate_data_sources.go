// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	shellquote "github.com/kballard/go-shellquote"
	"github.com/sanity-io/litter"
)

// This script is used to generate data sources for this terraform provider
//
// Usage:
// go run ../../scripts/generate_data_sources.go
// go run ../../scripts/generate_data_sources.go -resource auth-methods
// go run ../../scripts/generate_data_sources.go -resource credential-libraries
//
// This is tied to a target in the Makefile
// - make data-sources
// - Update provider.go to include the new data sources
// - make docs

var swaggerFile = "https://raw.githubusercontent.com/hashicorp/boundary/%s/internal/gen/controller.swagger.json"

const (
	NameKey     = "name"
	PageSizeKey = "page_size"
)

func main() {
	if err := Main(); err != nil {
		log.Fatal(err)
	}
}

func getBoundaryVersion() (string, error) {
	gomod, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(gomod), "\n") {
		if strings.Contains(line, "github.com/hashicorp/boundary") {
			parts := strings.Split(line, " ")
			if len(parts) < 2 {
				return "", fmt.Errorf("unexpected go.mod format")
			}
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("boundary version not found in go.mod")
}

func loadSwaggerFile(version string) (*spec.Swagger, error) {
	document, err := loads.JSONSpec(fmt.Sprintf(swaggerFile, version))
	if err != nil {
		return nil, fmt.Errorf("failed to load spec: %w", err)
	}

	swagger := document.Spec()
	// ExpandSpec is not careful to merge the attributes so we lose some
	// descriptions here
	if err = spec.ExpandSpec(swagger, nil); err != nil {
		return nil, fmt.Errorf("failed to expand spec")
	}

	return swagger, nil
}

func Main() error {
	// Optional flag to specify a single resource to generate
	var r string
	flag.StringVar(&r, "resource", "", "")
	flag.Parse()

	version, err := getBoundaryVersion()
	if err != nil {
		return fmt.Errorf("failed to get boundary version: %w", err)
	}
	fmt.Printf("Using Boundary version %s...\n", version)

	swagger, err := loadSwaggerFile(version)
	if err != nil {
		return fmt.Errorf("failed to load swagger file: %w", err)
	}

	// Get all resources if no resource was provided in a flag
	var rs []string
	if r == "" {
		for path := range swagger.Paths.Paths {
			resourceName := strings.Split(path, "/")[2]
			log.Printf("Resource: %s", resourceName)
			if !resourceExist(resourceName) {
				continue
			}
			if strings.HasSuffix(path, "{id}") {
				rs = append(rs, strings.Split(path, "/")[2])
			}
		}
	} else {
		rs = append(rs, r)
	}

	// Process resources
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	for _, r := range rs {
		fmt.Printf("Processing resource: %s...\n", r)
		resource, err := NewResourceFromSwagger(swagger, "/v1/", r)
		if err != nil {
			return err
		}

		source, err := resource.RenderPlural()
		if err != nil {
			return err
		}
		err = write(source, filepath.Join(basepath, "..", fmt.Sprintf(
			"internal/provider/data_source_%s.go",
			strings.Replace(resource.path, "-", "_", -1),
		)))
		if err != nil {
			return err
		}

		source, err = resource.RenderExamplePlural()
		if err != nil {
			return err
		}
		err = write(source, filepath.Join(basepath, "..", fmt.Sprintf(
			"examples/data-sources/boundary_%s/data-source.tf",
			strings.Replace(resource.path, "-", "_", -1),
		)))
		if err != nil {
			return err
		}

		// source, err = resource.RenderSingle()
		// if err != nil {
		// 	return err
		// }
		// err = write(source, filepath.Join(basepath, "..", fmt.Sprintf(
		// 	"internal/provider/data_source_%s.go",
		// 	strings.TrimSuffix(strings.Replace(resource.path, "-", "_", -1), "s"),
		// )))
		// if err != nil {
		// 	return err
		// }
	}

	return nil
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
	listParam   string
	schema      Schema
}

func NewResourceFromSwagger(swagger *spec.Swagger, root, path string) (*Resource, error) {
	sch := Schema{}
	if _, ok := swagger.Paths.Paths[root+path]; !ok {
		return nil, fmt.Errorf("path %q not found", root+path)
	}
	op := swagger.Paths.Paths[root+path].Get

	// Go through input parameters for getting the resource
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

	// Go through output properties of the resource
	for name, prop := range op.Responses.StatusCodeResponses[200].Schema.Properties {
		// If property is also a parameter, update existing schema
		if _, ok := sch[name]; ok {
			sch[name].Computed = true
			continue
		}

		s, err := getSchemaFromProperty(name, prop)
		if err != nil {
			return nil, err
		}
		if s == nil {
			continue
		}
		sch[name] = s
	}

	// Determine resourceName from path
	parts := strings.Split(path, "-")
	for i, part := range parts {
		parts[i] = cases.Title(language.English, cases.Compact).String(part)
	}
	resourceName := strings.Join(parts, "")

	// Find the field that is used to list the items for this resource. This
	// is done by finding a parameter that is a also a field in the items list
	// Example: For auth methods, you need the scope id
	var listParam string
	for name := range sch["items"].Elem.(*schema.Resource).Schema {
		if _, ok := sch[name]; ok {
			listParam = name
			break
		}
	}

	return &Resource{
		name:        resourceName, // Example: AuthMethods
		path:        path,         // Example: auth-methods
		listParam:   listParam,    // Example: scope_id
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
	case "integer":
		s.Type = schema.TypeInt
	default:
		return nil, fmt.Errorf("unknown type %q for %s", ty, p.Name)
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
	case "string":
		s.Type = schema.TypeString
	case "integer":
		s.Type = schema.TypeInt
	case "boolean":
		s.Type = schema.TypeBool
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
		return nil, fmt.Errorf("unknown type %q for %s", ty, name)
	}

	return s, nil
}

func (r *Resource) RenderExamplePlural() (string, error) {
	var example string
	if r.listParam == "" {
		example = fmt.Sprintf(`# Retrieve %s with "test" in the name
data "boundary_%s" "example" {
	filter = "\"test\" in \"/item/name\""
}
`,
			r.name,
			strings.Replace(r.path, "-", "_", -1),
		)
	} else {
		example = fmt.Sprintf(`# Retrieve %s
data "boundary_%s" "example" {
	%s = "id"
}

# Retrieve %s with "test" in the name
data "boundary_%s" "example" {
	filter = "\"test\" in \"/item/name\""
	%s = "id"
}
`,
			r.name,
			strings.Replace(r.path, "-", "_", -1),
			r.listParam,
			r.name,
			strings.Replace(r.path, "-", "_", -1),
			r.listParam,
		)
	}

	return example, nil
}

func (r *Resource) RenderSingle() (string, error) {
	// Find all input parameters from the items schema
	items := make(map[string]*schema.Schema)
	for k, v := range r.schema["items"].Elem.(*schema.Resource).Schema {
		items[k] = v
	}
	if r.listParam != "" {
		items[r.listParam].Optional = true
	}
	if _, ok := items[NameKey]; ok {
		items[NameKey].Optional = true
	}
	i := &Resource{
		schema: items,
	}

	attrs := map[string]schema.ValueType{}
	for name, schema := range i.schema {
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

	// For each input parameter, construct a query parameter for it
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
		case schema.TypeInt:
			imports["strconv"] = struct{}{}
			queryParams = append(queryParams, fmt.Sprintf(`q.Add(%q, strconv.Itoa(d.Get(%q).(int)))`, attr, attr))
		default:
			panic(fmt.Sprintf("unknown type %q for %q", ty, attr))
		}
	}

	imports_ := []string{}
	for i := range imports {
		imports_ = append(imports_, i)
	}
	sort.Strings(imports_)

	t := template.Must(template.New("datasource_single").Parse(`
// Code generated by "make datasources"; DO NOT EDIT.
// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
		Description: "Gets {{.Path}}",
		ReadContext: dataSource{{.Name}}Read,
		Schema: dataSource{{.Name}}Schema,
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

	d.SetId("asdfasdf")

	return nil
}

`))

	// !! q.Add("filter", fmt.Sprintf("\"/item/name\" == \"%s\"", d.Get("name").(string)))
	// !! set id to id of item
	// !! map other attributes correctly

	w := bytes.NewBuffer(nil)
	data := map[string]interface{}{
		"GenerateLine": fmt.Sprintf("generate go run ../../scripts/generate_data_sources.go %s", shellquote.Join("-name", r.name, "-path", r.path)),
		"Imports":      imports_,
		"Name":         strings.TrimSuffix(r.name, "s"), // Example: AuthMethod
		"Package":      strings.ToLower(r.name),         // Example: authmethods
		"Path":         strings.TrimSuffix(r.path, "s"), // Example: auth-method
		"Schema":       template.HTML(i.schema.String()),
		"Description":  template.HTML(fmt.Sprintf("%q", r.description)),
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

func (r *Resource) RenderPlural() (string, error) {
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
			if attr == r.listParam {
				queryParams = append(queryParams, fmt.Sprintf(`if d.Get(%q) != "" {`, r.listParam))
			}
			queryParams = append(queryParams, fmt.Sprintf(`q.Add(%q, d.Get(%q).(string))`, attr, attr))
			if attr == r.listParam {
				queryParams = append(queryParams, `}`)
			}
		case schema.TypeInt:
			imports["strconv"] = struct{}{}
			if attr == PageSizeKey {
				queryParams = append(queryParams, fmt.Sprintf(`if d.Get(%q) != 0 {`, r.listParam))
			}
			queryParams = append(queryParams, fmt.Sprintf(`q.Add(%q, strconv.Itoa(d.Get(%q).(int)))`, attr, attr))
			if attr == PageSizeKey {
				queryParams = append(queryParams, `}`)
			}
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
// Code generated by "make datasources"; DO NOT EDIT.
// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
		Description: "Lists {{.Path}}",
		ReadContext: dataSource{{.Name}}Read,
		Schema: dataSource{{.Name}}Schema,
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
		"GenerateLine": fmt.Sprintf("generate go run ../../scripts/generate_data_sources.go %s", shellquote.Join("-name", r.name, "-path", r.path)),
		"Imports":      imports_,
		"Name":         r.name,                // Example: AuthMethods
		"Path":         template.HTML(r.path), // Example: auth-methods
		"Schema":       template.HTML(r.schema.String()),
		"Description":  template.HTML(fmt.Sprintf("%q", r.description)),
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

func write(data string, filename string) error {
	err := os.MkdirAll(filepath.Dir(filename), os.ModePerm)
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write([]byte(data))
	return nil
}

// Check if terraform resource exists
func resourceExist(rs string) bool {
	// Format resource properly to match the file name
	name := strings.ReplaceAll(rs, "-", "_")

	if !strings.HasPrefix(name, "credentials") {
		name = strings.TrimSuffix(name, "s")
	}

	resourceFilePath := fmt.Sprintf("internal/provider/resource_%s*.go", name)

	// Assuming the resource files are in the internal/provider directory
	// Utilize glob as some resources are not matched 1-1 such as credential_stores data sources vs credential_store_static resource
	matches, err := filepath.Glob(resourceFilePath)
	if err != nil || len(matches) == 0 {
		return false
	}
	return true
}
