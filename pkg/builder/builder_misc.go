package builder

import (
	"strings"

	"github.com/henderiw/apiserver-builder/pkg/apiserver"
	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder/options"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	scheme "k8s.io/client-go/kubernetes/scheme"
	openapicommon "k8s.io/kube-openapi/pkg/common"
	apiextensionsopenapi "k8s.io/apiextensions-apiserver/pkg/generated/openapi"
)

func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {
		
	mergedDefs := func(ref openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
		result := apiextensionsopenapi.GetOpenAPIDefinitions(ref)
		for k, v := range defs(ref) {
			result[k] = v
		}

		// Add ~1-encoded aliases alongside the original keys.
		// getResourceNamesForGroup looks up by decoded "/" keys,
		// but NewTypeConverter resolves $refs using ~1-encoded keys — need both.
		aliases := make(map[string]openapicommon.OpenAPIDefinition)
		for k, v := range result {
			encodedKey := strings.ReplaceAll(k, "/", "~1")
			if encodedKey != k {
				aliases[encodedKey] = v
			}
		}
		for k, v := range aliases {
			result[k] = v
		}
		return result
	}

	namer := openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme)

		
	options.RecommendedConfigFns = append(options.RecommendedConfigFns, func(config *server.RecommendedConfig) *server.RecommendedConfig {
		config.OpenAPIConfig = server.DefaultOpenAPIConfig(mergedDefs, namer)
		config.OpenAPIConfig.Info.Title = name
		config.OpenAPIConfig.Info.Version = version
		config.OpenAPIV3Config = server.DefaultOpenAPIV3Config(mergedDefs, namer)
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
