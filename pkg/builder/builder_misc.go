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
	"k8s.io/kube-openapi/pkg/validation/spec"
)

func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {
		
	mergedDefs := func(ref openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
		// Wrap ref callback to decode ~1 → / so refs match map keys.
		// kube-openapi encodes "/" as "~1" in $ref strings for OpenAPI v3,
		// but StaticOpenAPISpec keys use raw "/" — NewTypeConverter can't resolve them.
		decodedRef := func(path string) spec.Ref {
			r := ref(path)
			refStr := r.String()
			decoded := strings.ReplaceAll(refStr, "~1", "/")
			if decoded != refStr {
				if newRef, err := spec.NewRef(decoded); err == nil {
					return newRef
				}
			}
			return r
		}
		result := apiextensionsopenapi.GetOpenAPIDefinitions(decodedRef)
		for k, v := range defs(decodedRef) {
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
