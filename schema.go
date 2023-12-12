package weave

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type Templates struct {
	ContentBlocks []*ContentBlock `hcl:"content,block"`
	DataBlocks    []*DataBlock    `hcl:"data,block"`
	Documents     []*Document     `hcl:"document,block"`
}

type MapTemplates struct {
	ContentBlocks map[string]*ContentBlock
	DataBlocks    map[string]*DataBlock
	Documents     map[string]*Document
}

type MetaBlock struct {
	Name        *string   `hcl:"name,optional"`
	Author      *string   `hcl:"author,optional"`
	Description *string   `hcl:"description,optional"`
	Tags        []*string `hcl:"tags,optional"`
	UpdatedAt   *string   `hcl:"updated_at,optional"`

	RequiredFields []*string `hcl:"required_fields,optional"`
}

type ContentBlock struct {
	Type string `hcl:"type,label"`
	Name string `hcl:"name,label"`

	Meta  *MetaBlock `hcl:"meta,block"`
	Query *string    `hcl:"query,optional"`
	Title *string    `hcl:"title,optional"`

	Rest hcl.Body `hcl:",remain"`

	// Parameters might contain Ref and ContentBlocks parts.
	// Ref field must point to another ContentBlock (referenced by name in the template
	// and resolved by HCL parser) and ContentBlocks contains subblocks in the current block
	//
	// Ref           ContentBlock   `hcl:"ref,optional"`
	// ContentBlocks []ContentBlock `hcl:"content,block"`
	//
	// https://hcl.readthedocs.io/en/latest/go_patterns.html#interdependent-blocks
}

type DataBlock struct {
	Type string `hcl:"type,label"`
	Name string `hcl:"type,label"`

	Meta *MetaBlock `hcl:"meta,block"`

	Rest hcl.Body `hcl:",remain"`

	// Rest might contain Ref field, that has another DataBlock,
	// referenced by name and resolved by HCL parser
	//
	// Ref           DataBlock   `hcl:"ref,optional"`
	//
	// https://hcl.readthedocs.io/en/latest/go_patterns.html#interdependent-blocks
}

type Document struct {
	Name string `hcl:"name,label"`

	Meta  *MetaBlock `hcl:"meta,block"`
	Title *string    `hcl:"title,optional"`

	DataBlocks    []*DataBlock    `hcl:"data,block"`
	ContentBlocks []*ContentBlock `hcl:"content,block"`
}

func (t *Templates) Map() *MapTemplates {
	m := &MapTemplates{
		ContentBlocks: make(map[string]*ContentBlock),
		DataBlocks:    make(map[string]*DataBlock),
		Documents:     make(map[string]*Document),
	}
	for _, cb := range t.ContentBlocks {
		m.ContentBlocks[fmt.Sprint("content.", cb.Type, ".", cb.Name)] = cb
	}
	for _, db := range t.DataBlocks {
		m.DataBlocks[fmt.Sprint("data.", db.Type, ".", db.Name)] = db
	}
	for _, doc := range t.Documents {
		m.Documents[fmt.Sprint("document.", doc.Name)] = doc
	}
	return m
}

func (t *MapTemplates) ProcessDoc(doc *Document) {
	for _, content := range doc.ContentBlocks {
		t.ProcessContent(content)
	}
	for _, data := range doc.DataBlocks {
		t.ProcessData(data)
	}
}

func (t *MapTemplates) ProcessData(data *DataBlock) {
	rest := &Templates{}
	if data.Type == "ref" {
		diag := gohcl.DecodeBody(data.Rest, nil, rest)
		if diag.HasErrors() {
			log.Println(diag.Error())
		}
	}
}

func ProcessReference(contentToProcess, referencedContent *ContentBlock) {
	refAttrs, err := contentToProcess.Rest.JustAttributes()
	if err != nil {
		log.Fatal(err)
	}
	newRest := fmt.Sprintf(
		"content %v %v {", referencedContent.Type, contentToProcess.Name,
	)
	for k, attr := range refAttrs {
		if k == "ref" {
			continue
		}
		val, err := attr.Expr.Value(nil)
		if err != nil {
			log.Fatal(err)
		}
		newRest += "\n" + k + " = " + `"` + val.AsString() + `"`
	}
	newRest += "\n}"
	f, err := hclsyntax.ParseConfig([]byte(newRest), "", hcl.Pos{Line: 1, Column: 1})
	if err != nil {
		log.Fatal(err)
	}
	type T struct {
		Cb *ContentBlock `hcl:"content,block"`
	}
	updatedContent := &ContentBlock{}
	err = gohcl.DecodeBody(f.Body, nil, &T{Cb: updatedContent})
	if err != nil {
		log.Fatal(updatedContent)
	}
	*contentToProcess = *updatedContent
}

func (t *MapTemplates) ProcessContent(contentToProcess *ContentBlock) {
	isRef := contentToProcess.Type == "ref"
	if true {
		contentAttrs, err := contentToProcess.Rest.JustAttributes()
		if err.HasErrors() {
			log.Fatal(err)
		}
		for k, contentAttr := range contentAttrs {
			if k == "ref" {
				if !isRef {
					ProcessReference(contentToProcess, contentToProcess)
					return
				}
				tr, ok := contentAttr.Expr.(*hclsyntax.ScopeTraversalExpr)
				if !ok {
					return
				}
				dotNotation := ""
				for _, v := range tr.Traversal {
					named, ok := v.(hcl.TraverseRoot)
					if ok {
						dotNotation += named.Name
						dotNotation += "."
					}
					named2, ok := v.(hcl.TraverseAttr)
					if ok {
						dotNotation += named2.Name
						dotNotation += "."
					}
				}
				dotNotation = dotNotation[0 : len(dotNotation)-1]
				referencedContent, ok := t.ContentBlocks[dotNotation]

				if ok {
					ProcessReference(contentToProcess, referencedContent)
				}
			}
		}
	}
}
