package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	flag "github.com/ondbyte/turbo_flag"
	"github.com/ondbyte/weave"
)

func parseTemplates(filename string) weave.Templates {
	var templates weave.Templates
	var file *hcl.File
	var diags hcl.Diagnostics

	src, _ := ioutil.ReadFile(filename)

	file, diags = hclsyntax.ParseConfig(src, filename, hcl.Pos{Line: 1, Column: 1})
	diags = gohcl.DecodeBody(file.Body, nil, &templates)

	if diags.HasErrors() {
		log.Fatalf("Failed to load configuration: %s", diags)
	}

	return templates
}

func main() {
	flag.MainCmd("weave", "use to parse the hcl file", flag.ExitOnError, os.Args, WeaveCli)
}

func WeaveCli(cmd flag.CMD, args []string) {
	log.SetFlags(log.Llongfile)
	path := ""
	document := ""
	help := false
	cmd.StringVar(&path, "path", "", "path of the dir containing *.hcl files", flag.Alias("p"))
	cmd.StringVar(&document, "document", "", "name of the document to t.process", flag.Alias("d"))
	cmd.BoolVar(&help, "help", false, "help", flag.Alias("h"))
	err := cmd.Parse(args[1:])
	if err != nil {
		log.Println(cmd.GetDefaultUsageLong())
		panic(err)
	}
	if help {
		log.Println(cmd.GetDefaultUsageLong())
		os.Exit(0)
	}
	if path == "" {
		log.Panic("path parameter is required")
	}
	if document == "" {
		log.Panic("document parameter is required")
	}
	dirs, err := os.ReadDir(path)
	if err != nil {
		log.Panic(err)
	}
	n := 0
	t := &weave.Templates{
		ContentBlocks: []*weave.ContentBlock{},
		DataBlocks:    []*weave.DataBlock{},
		Documents:     []*weave.Document{},
	}

	for _, e := range dirs {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".hcl") {
			n++
			t2 := parseTemplates(filepath.Join(path, e.Name()))
			t.ContentBlocks = append(t.ContentBlocks, t2.ContentBlocks...)
			t.DataBlocks = append(t.DataBlocks, t2.DataBlocks...)
			t.Documents = append(t.Documents, t2.Documents...)
		}
	}
	log.Printf("read %v hcl file", n)
	m := t.Map()
	for _, content := range t.ContentBlocks {
		err := m.ProcessContent(content)
		if err != nil {
			log.Panic(err)
		}
	}
	for _, data := range m.DataBlocks {
		err := m.ProcessData(data)
		if err != nil {
			log.Panic(err)
		}
	}
	for _, doc := range m.Documents {
		err := m.ProcessDoc(doc)
		if err != nil {
			log.Panic(err)
		}
	}
	selectedDocument, ok := m.Documents[fmt.Sprintf("document.%v", document)]
	if !ok {
		log.Fatalf("selected doc %v doesnt exists in hcl source\n", document)
	}
	log.Println(selectedDocument)
	//start the plugins
	pluginA, pluginClient, err := weave.LoadPlugin("../plugins/plugin_a/plugin_a")
	if err != nil {
		log.Panic(err)
	}
	defer pluginClient.Kill()

	storage := map[string]map[string]map[string]interface{}{"data": map[string]map[string]interface{}{}}

	for _, dataBlock := range selectedDocument.DataBlocks {
		// plugin calls are not concurrent
		switch dataBlock.Type {
		case "plugin_a":
			{
				attrs, diag := dataBlock.Rest.JustAttributes()
				if diag != nil {
					log.Panic(diag)
				}
				m := map[string]interface{}{}
				for k, attr := range attrs {
					val, err := attr.Expr.Value(nil)
					if err != nil {
						log.Panic(err)
					}
					i, _ := val.AsBigFloat().Int64()
					m[k] = i
				}
				response, err := pluginA.Process(m)
				if err != nil {
					log.Panic(err)
				}
				result, ok := response["result"].(int64)
				if !ok {
					log.Fatalf(`response should have a key "result" with value of type int64 `+
						"which is a sum of the parameters in map=%v\n", m)
				}
				typeStorage, ok := storage["data"][dataBlock.Type]
				if !ok {
					typeStorage = map[string]interface{}{}
					storage["data"][dataBlock.Type] = typeStorage
				}
				typeStorage[dataBlock.Name] = result
			}
		}
	}

	toPrint, err := json.MarshalIndent(storage, "", "	")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("\n=====output=======\n" + string(toPrint) + "\n==============")
}
