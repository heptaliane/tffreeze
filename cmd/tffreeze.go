package main

import (
	"flag"
	"log"
	"os"
	"regexp"
    "reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type CommandLineArgument struct {
	varfile string
	tffiles []string
}

func loadHclsyntaxFile(path string) (*hcl.File, hcl.Diagnostics) {
	content, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return hclsyntax.ParseConfig(content, path, hcl.Pos{})
}

func loadHclwriteFile(path string) (*hclwrite.File, hcl.Diagnostics) {
	content, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return hclwrite.ParseConfig(content, path, hcl.Pos{})
}

func loadTfvars(path string) (map[string]cty.Value, hcl.Diagnostics) {
	file, diags := loadHclsyntaxFile(path)
	if diags.HasErrors() {
		return nil, diags
	}

	attrs, attrDiags := file.Body.JustAttributes()
	diags = append(diags, attrDiags...)
	if attrDiags.HasErrors() {
		return nil, diags
	}

	values := make(map[string]cty.Value)
	for key, attr := range attrs {
		value, valueDiags := attr.Expr.Value(nil)
		diags = append(diags, valueDiags...)
		values[key] = value
	}

	return values, diags
}

func substituteExpression(expr hclsyntax.Expression, ctx *hcl.EvalContext) (*hclsyntax.Expression, hcl.Diagnostics) {
	value, diags := expr.Value(ctx)
	if diags.HasErrors() {
		return nil, diags
	}

	var subExpr hclsyntax.Expression
	subExpr = &hclsyntax.LiteralValueExpr{
		Val:      value,
		SrcRange: expr.StartRange(),
	}
	return &subExpr, diags
}

func substituteVariables(body *hclsyntax.Body, subBody *hclwrite.Body, ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	for name, attr := range body.Attributes {
		value, attrDiags := attr.Expr.Value(ctx)
		if !attrDiags.HasErrors() {
			diags = append(diags, attrDiags...)
			subBody.SetAttributeValue(name, value)
		}
	}

	for i := range body.Blocks {
		substituteVariables(body.Blocks[i].Body, subBody.Blocks()[i].Body(), ctx)
	}

	return diags
}

func substituteVariablesRecursive(body *hclsyntax.Body, subBody *hclwrite.Body, ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}
	bodies := []*hclsyntax.Body{body}
	subBodies := []*hclwrite.Body{subBody}

	for {
		newBodies := []*hclsyntax.Body{}
		newSubBodies := []*hclwrite.Body{}
		for i := range bodies {
			subDiags := substituteVariables(bodies[i], subBodies[i], ctx)
			diags = append(diags, subDiags...)

			blocks := bodies[i].Blocks
			subBlocks := subBodies[i].Blocks()
			for j := range blocks {
				newBodies = append(newBodies, blocks[j].Body)
				newSubBodies = append(newSubBodies, subBlocks[j].Body())
			}
		}

		bodies = newBodies
		subBodies = newSubBodies

		if len(bodies) == 0 {
			return diags
		}
	}
}

func parseCommandLineArgument() CommandLineArgument {
	args := CommandLineArgument{}

	flag.StringVar(&args.varfile, "var-file", "", "Path to variable definitions file")
	flag.Parse()

	args.tffiles = flag.Args()

	return args
}

func main() {
	args := parseCommandLineArgument()

	values := make(map[string]cty.Value)
	if _, err := os.Stat(args.varfile); err == nil {
		filevals, diags := loadTfvars(args.varfile)
		if diags.HasErrors() {
			log.Fatal(diags.Error())
		}
		for key, value := range filevals {
			values[key] = value
		}
	}
	ctx := hcl.EvalContext{
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(values),
		},
	}

	dstfileRegexp := regexp.MustCompile("(\\.[^\\.]*)$")

	for _, path := range args.tffiles {
		hclsyntaxFile, diags := loadHclsyntaxFile(path)
		if diags.HasErrors() {
			log.Println(diags.Error())
			continue
		}

		hclwriteFile, diags := loadHclwriteFile(path)
		if diags.HasErrors() {
			log.Println(diags.Error())
			continue
		}

		substituteVariablesRecursive(hclsyntaxFile.Body.(*hclsyntax.Body), hclwriteFile.Body(), &ctx)

		dstfilename := dstfileRegexp.ReplaceAllString(path, ".freeze${1}")
		err := os.WriteFile(dstfilename, hclwriteFile.Bytes(), 0644)
		if err == nil {
			log.Printf("Variable freezed: \"%s\" -> \"%s\"", path, dstfilename)
		} else {
			log.Println(err)
		}
	}
}
