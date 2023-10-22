package main

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"testing"
)

const TEXT_VARIABLES string = `
a = 1
b = true
# c = "commented"
d = "annotated"  # Some comment
e = ["arr1", "arr2"]
f = {
  item1 = "value1"
  item2 = "value2"
  item3 = 0
}
g = {
  h = {
    i = "foo"
    j = "bar"
  }
}
`

func getExampleEvalContext() hcl.EvalContext {
	file, diags := hclsyntax.ParseConfig([]byte(TEXT_VARIABLES), "", hcl.Pos{})
	if diags.HasErrors() {
		panic(diags.Error())
	}

	attrs, diags := file.Body.JustAttributes()
	if diags.HasErrors() {
		panic(diags.Error())
	}

	values := make(map[string]cty.Value)
	for key, attr := range attrs {
		value, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			panic(diags.Error())
		}
		values[key] = value
	}

	return hcl.EvalContext{
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(values),
		},
	}
}

func parseHCL(text string) (*hcl.File, *hclwrite.File) {
	hclsyntaxFile, diags := hclsyntax.ParseConfig([]byte(text), "", hcl.Pos{})
	if diags.HasErrors() {
		panic(diags.Error())
	}

	hclwriteFile, diags := hclwrite.ParseConfig([]byte(text), "", hcl.Pos{})
	if diags.HasErrors() {
		panic(diags.Error())
	}
	return hclsyntaxFile, hclwriteFile
}

func TestSubstituteVariables(t *testing.T) {
	ctx := getExampleEvalContext()

	bodies := []string{
		"A = var.a",
		"B = var.b",
		"C = var.c",
		"D = var.d",
		"E = var.e",
		"F = var.f",
		"G = var.g",
		"H = var.g.h",
		"AB = \"${var.a}-${var.b}\"",
		"A_B = \"${var.a}-var.b\"",
		"ABX = \"${var.a}-${var.b}-${var.x}\"",
		"map_F = [ for key, value in var.f : value ]",
		"func_A = length(var.e)",
	}
	expected := []string{
		"A = 1",
		"B = true",
		"C = var.c",
		"D = \"annotated\"",
		"E = [\"arr1\", \"arr2\"]",
		"F = {\n" +
			"  item1 = \"value1\"\n" +
			"  item2 = \"value2\"\n" +
			"  item3 = 0\n" +
			"}",
		"G = {\n" +
			"  h = {\n" +
			"    i = \"foo\"\n" +
			"    j = \"bar\"\n" +
			"  }\n" +
			"}",
		"H = {\n" +
			"  i = \"foo\"\n" +
			"  j = \"bar\"\n" +
			"}",
		"AB = \"1-true\"",
		"A_B = \"1-var.b\"",
		"ABX = \"1-true-${var.x}\"",
		"map_F = [\"value1\", \"value2\", 0]",
		"func_A = length([\"arr1\", \"arr2\"])",
	}

	for i := range bodies {
		sfile, wfile := parseHCL(bodies[i])
		diags := substituteVariables(sfile.Body.(*hclsyntax.Body), wfile.Body(), &ctx)
		if diags.HasErrors() {
			t.Error(diags.Error())
		}

		t.Run(bodies[i], func(t *testing.T) {
			actual := string(wfile.Bytes())
			if actual != expected[i] {
				t.Errorf("\nActual:\n%s\n---\nExpected:\n%s", actual, expected[i])
				t.Errorf("Diags: %s", diags)
			}
		})
	}
}
