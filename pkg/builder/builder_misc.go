package builder

import (
	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	scheme "k8s.io/client-go/kubernetes/scheme"
	openapicommon "k8s.io/kube-openapi/pkg/common"
)

func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {
	// set both openAPI definitions
	apiserverbuilder.RecommendedConfigFns = append(apiserverbuilder.RecommendedConfigFns, func(config *server.RecommendedConfig) *server.RecommendedConfig {
		config.OpenAPIConfig = server.DefaultOpenAPIConfig(defs, openapinamer.NewDefinitionNamer(apiserverbuilder.Scheme, scheme.Scheme))
		config.OpenAPIConfig.Info.Title = name
		config.OpenAPIConfig.Info.Version = version
		config.OpenAPIV3Config = server.DefaultOpenAPIV3Config(defs, openapinamer.NewDefinitionNamer(apiserverbuilder.Scheme, scheme.Scheme))
		config.OpenAPIV3Config.Info.Title = name
		config.OpenAPIV3Config.Info.Version = version
		return config
	})
	return r
}

// WithoutEtcd removes etcd related settings from apiserver.
func (r *Server) WithoutEtcd() *Server {
	return r.WithOptionsFns(func(o *ServerOptions) *ServerOptions {
		o.RecommendedOptions.Etcd = nil
		return o
	})
}

func (r *Server) WithServerName(serverName string) *Server {
	r.ServerName = serverName
	return r
}

func (r *Server) WithEtcdPath(etcdPath string) *Server {
	r.EtcdPath = etcdPath
	return r
}
