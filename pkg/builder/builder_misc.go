package builder

import (
	"fmt"
	
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
		return result
	}
	
	// DEBUG: check what the namer returns for our types
    namer := openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme)
    defName, _ := namer.GetDefinitionName("github.com/sdcio/config-server/apis/config/v1alpha1.Config")
    fmt.Printf("DEBUG: namer Config = %q\n", defName)
    defName2, _ := namer.GetDefinitionName("github.com/sdcio/config-server/apis/config/v1alpha1.DeviationSpec")
    fmt.Printf("DEBUG: namer DeviationSpec = %q\n", defName2)
		
	options.RecommendedConfigFns = append(options.RecommendedConfigFns, func(config *server.RecommendedConfig) *server.RecommendedConfig {
		config.OpenAPIConfig = server.DefaultOpenAPIConfig(mergedDefs, openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme))
		config.OpenAPIConfig.Info.Title = name
		config.OpenAPIConfig.Info.Version = version
		config.OpenAPIV3Config = server.DefaultOpenAPIV3Config(mergedDefs, openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme))
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
