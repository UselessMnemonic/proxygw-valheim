package valheim

import (
	"github.com/UselessMnemonic/proxygw-valheim/frontends"
	"github.com/UselessMnemonic/proxygw/pkg/engine"
	"github.com/UselessMnemonic/proxygw/plugin"
)

func init() {
	plugin.Register("github.com/UselessMnemonic/proxygw-valheim", plugin.Handler{
		OnLoad: func(_ map[string]any, _ *engine.Engine, namespace *plugin.Namespace) error {
			namespace.Frontends["server"] = frontends.NewServerHandler
			namespace.Frontends["status"] = frontends.NewA2SHandler
			return nil
		},
		OnUnload: func() error {
			return nil
		},
	})
}
