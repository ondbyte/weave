// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"

	"github.com/hashicorp/go-plugin"
	"github.com/ondbyte/weave"
)

// Here is a real implementation of Greeter
type PluginA struct {
}

func New() weave.Plugin {
	return &PluginA{}
}

// Process implements weave.Plugin.
func (*PluginA) Process(msg map[string]interface{}) (map[string]interface{}, error) {
	a, ok := msg["parameter_x"].(int64)
	if !ok {
		return nil, fmt.Errorf(`value of param "parameter_x" as a int64 is required`)
	}
	b, ok := msg["parameter_y"].(int64)
	if !ok {
		return nil, fmt.Errorf(`value of param "parameter_y" as a int64 is required`)
	}
	return map[string]interface{}{"result": a + b}, nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: weave.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			weave.PluginId: &weave.PluginWrapper{PluginImplementation: New()},
		},
	})
}
