package weave

import (
	"net/rpc"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

const PluginId = "weave_plugin"

type Plugin interface {
	Process(msg map[string]interface{}) (map[string]interface{}, error)
}

// Here is an implementation that talks over RPC
type WeavePluginRPCClient struct{ client *rpc.Client }

func NewRpcClient(c *rpc.Client) Plugin {
	return &WeavePluginRPCClient{
		client: c,
	}
}
func (g *WeavePluginRPCClient) Process(msg map[string]interface{}) (map[string]interface{}, error) {
	var resp = map[string]interface{}{}
	err := g.client.Call("Plugin.Process", msg, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type WeavePluginRPCServer struct {
	Impl Plugin
}

func (s *WeavePluginRPCServer) Process(msg map[string]interface{}, resp *map[string]interface{}) error {
	r, err := s.Impl.Process(msg)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

type PluginWrapper struct {
	PluginImplementation Plugin
}

func (p *PluginWrapper) Server(*plugin.MuxBroker) (interface{}, error) {
	return &WeavePluginRPCServer{Impl: p.PluginImplementation}, nil
}

func (PluginWrapper) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return NewRpcClient(c), nil
}

func LoadPlugin(path string) (Plugin, *plugin.Client, error) {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   PluginId,
		Output: os.Stdout,
		Level:  hclog.Debug,
	})

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			PluginId: &PluginWrapper{},
		},
		Cmd:    exec.Command(path),
		Logger: logger,
	})

	rpcClient, err := client.Client()
	if err != nil {
		return nil, nil, err
	}
	raw, err := rpcClient.Dispense(PluginId)
	if err != nil {
		return nil, nil, err
	}
	return raw.(Plugin), client, err

}

var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "weave",
	MagicCookieValue: "1234",
}
