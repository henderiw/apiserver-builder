package builder

import (
	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder"
	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/server"
)

// WithOptionsFns sets functions to customize the ServerOptions used to create the apiserver
func (r *Server) WithOptionsFns(fns ...func(*ServerOptions) *ServerOptions) *Server {
	apiserverbuilder.ServerOptionsFns = append(apiserverbuilder.ServerOptionsFns, fns...)
	return r
}

// WithServerFns sets functions to customize the GenericAPIServer
func (r *Server) WithServerFns(fns ...func(server *GenericAPIServer) *GenericAPIServer) *Server {
	apiserverbuilder.GenericAPIServerFns = append(apiserverbuilder.GenericAPIServerFns, fns...)
	return r
}

// WithConfigFns sets functions to customize the RecommendedConfig
func (r *Server) WithConfigFns(fns ...func(config *server.RecommendedConfig) *server.RecommendedConfig) *Server {
	apiserverbuilder.RecommendedConfigFns = append(apiserverbuilder.RecommendedConfigFns, fns...)
	return r
}

// WithFlagFns sets functions to customize the flags for the compiled binary.
func (r *Server) WithFlagFns(fns ...func(set *pflag.FlagSet) *pflag.FlagSet) *Server {
	apiserverbuilder.FlagsFns = append(apiserverbuilder.FlagsFns, fns...)
	return r
}
