package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var updateSnapshot = flag.Bool("update", false, "rewrite testdata/schema.json from the current provider schema")

// snapshot is the part of the provider schema that the semver promise
// covers: the provider block and every resource, data source, ephemeral
// resource, and function. The compact types below carry the schema, plus
// the defaults, plan modifiers, and validators of the framework schema.
// Descriptions are documentation, which the docs check guards, so a wording
// change leaves the snapshot untouched, and empty fields are omitted.
type snapshot struct {
	Provider           *snapSchema                    `json:"provider"`
	Resources          map[string]*snapSchema         `json:"resources"`
	DataSources        map[string]*snapSchema         `json:"data_sources"`
	EphemeralResources map[string]*snapSchema         `json:"ephemeral_resources,omitempty"`
	Functions          map[string]*tfprotov6.Function `json:"functions,omitempty"`
}

type snapSchema struct {
	Version int64      `json:"version"`
	Block   *snapBlock `json:"block"`
}

type snapBlock struct {
	Attributes []snapAttribute   `json:"attributes,omitempty"`
	Blocks     []snapNestedBlock `json:"blocks,omitempty"`
	Deprecated bool              `json:"deprecated,omitempty"`
}

type snapNestedBlock struct {
	TypeName string     `json:"type_name"`
	Nesting  string     `json:"nesting"`
	MinItems int64      `json:"min_items,omitempty"`
	MaxItems int64      `json:"max_items,omitempty"`
	Block    *snapBlock `json:"block"`
}

type snapAttribute struct {
	Name       string      `json:"name"`
	Type       string      `json:"type,omitempty"`
	Nested     *snapObject `json:"nested,omitempty"`
	Required   bool        `json:"required,omitempty"`
	Optional   bool        `json:"optional,omitempty"`
	Computed   bool        `json:"computed,omitempty"`
	Sensitive  bool        `json:"sensitive,omitempty"`
	WriteOnly  bool        `json:"write_only,omitempty"`
	Deprecated bool        `json:"deprecated,omitempty"`
	// The protocol schema does not carry these three. They come from the
	// framework schema: a changed default changes the plan of an existing
	// configuration, a new RequiresReplace replaces a resource, and a
	// tighter validator rejects a configuration that worked before.
	Default       string   `json:"default,omitempty"`
	PlanModifiers []string `json:"plan_modifiers,omitempty"`
	Validators    []string `json:"validators,omitempty"`
}

// attrExtra holds the framework-only part of one attribute.
type attrExtra struct {
	Default       string
	PlanModifiers []string
	Validators    []string
}

type snapObject struct {
	Nesting    string          `json:"nesting"`
	Attributes []snapAttribute `json:"attributes"`
}

// TestSchemaSnapshot pins the provider schema to testdata/schema.json. A
// schema change fails here until the snapshot is regenerated:
//
//	go test ./internal/provider -run TestSchemaSnapshot -update
//
// Review the diff of that file in the pull request. A removed attribute, a
// type change, an optional attribute that became required, a changed
// default, a new RequiresReplace, or a tighter validator is a breaking
// change and needs a major version; an added optional attribute needs a
// minor version.
func TestSchemaSnapshot(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(New("test")())()
	resp, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("schema diagnostic: %s: %s", d.Summary, d.Detail)
		}
	}
	provExtras, resExtras, dsExtras := frameworkExtras(ctx, New("test")())
	snap := snapshot{
		Provider:           normalize(resp.Provider, provExtras),
		Resources:          normalizeAll(resp.ResourceSchemas, resExtras),
		DataSources:        normalizeAll(resp.DataSourceSchemas, dsExtras),
		EphemeralResources: normalizeAll(resp.EphemeralResourceSchemas, nil),
		Functions:          resp.Functions,
	}
	got, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got = append(got, '\n')

	path := filepath.Join("testdata", "schema.json")
	if *updateSnapshot {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run with -update to create it)", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("the provider schema differs from %s.\n"+
			"Run `go test ./internal/provider -run TestSchemaSnapshot -update`, then review the diff:\n"+
			"a removed attribute, a type change, or an optional attribute that became required is a breaking change.", path)
	}
}

func normalizeAll(in map[string]*tfprotov6.Schema, extras map[string]map[string]attrExtra) map[string]*snapSchema {
	out := make(map[string]*snapSchema, len(in))
	for name, s := range in {
		out[name] = normalize(s, extras[name])
	}
	return out
}

// normalize sorts the attributes and blocks by name, so the JSON is stable.
func normalize(s *tfprotov6.Schema, extras map[string]attrExtra) *snapSchema {
	if s == nil {
		return nil
	}
	return &snapSchema{Version: s.Version, Block: normalizeBlock(s.Block, "", extras)}
}

func normalizeBlock(b *tfprotov6.SchemaBlock, prefix string, extras map[string]attrExtra) *snapBlock {
	if b == nil {
		return nil
	}
	out := &snapBlock{Deprecated: b.Deprecated}
	for _, a := range b.Attributes {
		out.Attributes = append(out.Attributes, normalizeAttribute(a, prefix, extras))
	}
	sort.Slice(out.Attributes, func(i, j int) bool { return out.Attributes[i].Name < out.Attributes[j].Name })
	for _, nb := range b.BlockTypes {
		out.Blocks = append(out.Blocks, snapNestedBlock{
			TypeName: nb.TypeName,
			Nesting:  fmt.Sprint(nb.Nesting),
			MinItems: nb.MinItems,
			MaxItems: nb.MaxItems,
			Block:    normalizeBlock(nb.Block, prefix+nb.TypeName+".", extras),
		})
	}
	sort.Slice(out.Blocks, func(i, j int) bool { return out.Blocks[i].TypeName < out.Blocks[j].TypeName })
	return out
}

func normalizeAttribute(a *tfprotov6.SchemaAttribute, prefix string, extras map[string]attrExtra) snapAttribute {
	ex := extras[prefix+a.Name]
	out := snapAttribute{
		Name:          a.Name,
		Required:      a.Required,
		Optional:      a.Optional,
		Computed:      a.Computed,
		Sensitive:     a.Sensitive,
		WriteOnly:     a.WriteOnly,
		Deprecated:    a.Deprecated,
		Default:       ex.Default,
		PlanModifiers: ex.PlanModifiers,
		Validators:    ex.Validators,
	}
	if a.Type != nil {
		out.Type = a.Type.String()
	}
	if a.NestedType != nil {
		nt := &snapObject{Nesting: fmt.Sprint(a.NestedType.Nesting)}
		for _, na := range a.NestedType.Attributes {
			nt.Attributes = append(nt.Attributes, normalizeAttribute(na, prefix+a.Name+".", extras))
		}
		sort.Slice(nt.Attributes, func(i, j int) bool { return nt.Attributes[i].Name < nt.Attributes[j].Name })
		out.Nested = nt
	}
	return out
}

// frameworkExtras reads the framework schema of the provider, of each
// resource, and of each data source, keyed by type name and then by the
// dotted attribute path.
func frameworkExtras(ctx context.Context, p provider.Provider) (prov map[string]attrExtra, res, ds map[string]map[string]attrExtra) {
	var md provider.MetadataResponse
	p.Metadata(ctx, provider.MetadataRequest{}, &md)

	var ps provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &ps)
	prov = map[string]attrExtra{}
	collectExtras(ctx, "", reflect.ValueOf(ps.Schema.Attributes), prov)

	res = map[string]map[string]attrExtra{}
	for _, f := range p.Resources(ctx) {
		r := f()
		var rm resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: md.TypeName}, &rm)
		var rs resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &rs)
		res[rm.TypeName] = map[string]attrExtra{}
		collectExtras(ctx, "", reflect.ValueOf(rs.Schema.Attributes), res[rm.TypeName])
	}

	ds = map[string]map[string]attrExtra{}
	for _, f := range p.DataSources(ctx) {
		d := f()
		var dm datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: md.TypeName}, &dm)
		var dsr datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &dsr)
		ds[dm.TypeName] = map[string]attrExtra{}
		collectExtras(ctx, "", reflect.ValueOf(dsr.Schema.Attributes), ds[dm.TypeName])
	}
	return prov, res, ds
}

// collectExtras walks a map of framework attributes. The attribute types
// differ per value type (StringDefaultValue, Int64PlanModifiers, and so
// on), so the walk finds the methods by their name suffix.
func collectExtras(ctx context.Context, prefix string, attrs reflect.Value, out map[string]attrExtra) {
	iter := attrs.MapRange()
	for iter.Next() {
		name := prefix + iter.Key().String()
		attr := iter.Value().Elem()
		var ex attrExtra
		t := attr.Type()
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			if m.Type.NumIn() != 1 {
				continue
			}
			switch {
			case strings.HasSuffix(m.Name, "DefaultValue"):
				if v := attr.Method(i).Call(nil)[0]; !v.IsNil() {
					ex.Default = defaultValue(ctx, v.Interface())
				}
			case strings.HasSuffix(m.Name, "PlanModifiers"):
				ex.PlanModifiers = describeAll(ctx, attr.Method(i).Call(nil)[0])
			case strings.HasSuffix(m.Name, "Validators"):
				ex.Validators = describeAll(ctx, attr.Method(i).Call(nil)[0])
			}
		}
		if ex.Default != "" || ex.PlanModifiers != nil || ex.Validators != nil {
			out[name] = ex
		}
		if nested := attr.FieldByName("NestedObject"); nested.IsValid() {
			collectExtras(ctx, name+".", nested.FieldByName("Attributes"), out)
		}
	}
}

func describeAll(ctx context.Context, list reflect.Value) []string {
	var out []string
	for i := 0; i < list.Len(); i++ {
		out = append(out, list.Index(i).Interface().(interface{ Description(context.Context) string }).Description(ctx))
	}
	return out
}

// defaultValue returns the value that a default sets, in the Terraform
// value syntax, for example "RS256" with its quotes or [] for an empty set.
func defaultValue(ctx context.Context, d any) string {
	switch d := d.(type) {
	case defaults.String:
		var r defaults.StringResponse
		d.DefaultString(ctx, defaults.StringRequest{}, &r)
		return r.PlanValue.String()
	case defaults.Int64:
		var r defaults.Int64Response
		d.DefaultInt64(ctx, defaults.Int64Request{}, &r)
		return r.PlanValue.String()
	case defaults.Bool:
		var r defaults.BoolResponse
		d.DefaultBool(ctx, defaults.BoolRequest{}, &r)
		return r.PlanValue.String()
	case defaults.Set:
		var r defaults.SetResponse
		d.DefaultSet(ctx, defaults.SetRequest{}, &r)
		return r.PlanValue.String()
	case defaults.List:
		var r defaults.ListResponse
		d.DefaultList(ctx, defaults.ListRequest{}, &r)
		return r.PlanValue.String()
	case defaults.Map:
		var r defaults.MapResponse
		d.DefaultMap(ctx, defaults.MapRequest{}, &r)
		return r.PlanValue.String()
	}
	return d.(interface{ Description(context.Context) string }).Description(ctx)
}
