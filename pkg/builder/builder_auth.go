package builder

import (
	"log/slog"

	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder/options"
	"github.com/spf13/pflag"
)

var enablesLocalStandaloneDebugging bool

// WithLocalDebugExtension adds an optional local-debug mode to the apiserver so that it can be tested
// locally without involving a complete kubernetes cluster. A flag named "--standalone-debug-mode" will
// also be added the binary which forcily requires "--bind-address" to be "127.0.0.1" in order to avoid
// security issues.
func (a *Server) WithLocalDebugExtension() *Server {
	options.ServerOptionsFns = append(options.ServerOptionsFns, func(options *ServerOptions) *ServerOptions {
		secureBindingAddr := options.RecommendedOptions.SecureServing.BindAddress.String()
		if enablesLocalStandaloneDebugging {
			if secureBindingAddr != "127.0.0.1" {
				slog.Error(`--bind-address must be "127.0.0.1" if --standalone-debug-mode is set`)
			}
			options.RecommendedOptions.Authorization = nil
			options.RecommendedOptions.CoreAPI = nil
			options.RecommendedOptions.Admission = nil
		}
		return options
	})
	options.FlagsFns = append(options.FlagsFns, func(fs *pflag.FlagSet) *pflag.FlagSet {
		fs.BoolVar(&enablesLocalStandaloneDebugging, "standalone-debug-mode", false,
			"Under the local-debug mode the apiserver will allow all access to its resources without "+
				"authorizing the requests, this flag is only intended for debugging in your workstation "+
				"and the apiserver will be crashing if its binding address is not 127.0.0.1.")
		return fs
	})
	options.ServerOptionsFns = append(options.ServerOptionsFns, func(o *ServerOptions) *ServerOptions {
		o.RecommendedOptions.Authentication.RemoteKubeConfigFileOptional = true
		return o
	})
	return a
}
