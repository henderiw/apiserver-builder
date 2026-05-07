package builder

import (
	"github.com/henderiw/apiserver-builder/pkg/apiserver"
	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder/options"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	scheme "k8s.io/client-go/kubernetes/scheme"
	apiextensionsopenapi "k8s.io/apiextensions-apiserver/pkg/generated/openapi"
	openapicommon "k8s.io/kube-openapi/pkg/common"
)

// WithOpenAPIDefinitions registers OpenAPI definitions for the API server.
//
// It uses openapinamer.NewDefinitionNamer to produce stable, reverse-DNS component
// names (e.g. com.github.sdcio.config-server.apis.config.v1alpha1.ConfigSpec) that
// are consistent across both OpenAPI v2 and v3, and that contain no characters
// requiring JSON Pointer escaping.  This means $ref values in the served spec are
// resolvable by kubectl's client-side validator without any double-encoding issues.
//
// The definitions are merged with the apiextensions-apiserver built-in types so that
// CRD-related and apimachinery types are always present in the schema.
func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {

	namer := openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme)

	mergedDefs := func(ref openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
		result := apiextensionsopenapi.GetOpenAPIDefinitions(ref)
		for k, v := range defs(ref) {
			result[k] = v
		}
		return result
	}

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