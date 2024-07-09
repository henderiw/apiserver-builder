package builder

import (
	"github.com/henderiw/apiserver-builder/pkg/apiserver"
	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder/options"
	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/server"
)

// WithOptionsFns sets functions to customize the ServerOptions used to create the apiserver
func (r *Server) WithOptionsFns(fns ...func(*ServerOptions) *ServerOptions) *Server {
	options.ServerOptionsFns = append(options.ServerOptionsFns, fns...)
	return r
}

// WithServerFns sets functions to customize the GenericAPIServer
func (r *Server) WithServerFns(fns ...func(server *GenericAPIServer) *GenericAPIServer) *Server {
	apiserver.GenericAPIServerFns = append(apiserver.GenericAPIServerFns, fns...)
	return r
}

// WithConfigFns sets functions to customize the RecommendedConfig
func (r *Server) WithConfigFns(fns ...func(config *server.RecommendedConfig) *server.RecommendedConfig) *Server {
	options.RecommendedConfigFns = append(options.RecommendedConfigFns, fns...)
	return r
}

// WithFlagFns sets functions to customize the flags for the compiled binary.
func (r *Server) WithFlagFns(fns ...func(set *pflag.FlagSet) *pflag.FlagSet) *Server {
	options.FlagsFns = append(options.FlagsFns, fns...)
	return r
}
