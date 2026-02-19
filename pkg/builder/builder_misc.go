package builder

import (
	"fmt"
	"strings"

	"github.com/henderiw/apiserver-builder/pkg/apiserver"
	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder/options"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	scheme "k8s.io/client-go/kubernetes/scheme"
	openapicommon "k8s.io/kube-openapi/pkg/common"
	apiextensionsopenapi "k8s.io/apiextensions-apiserver/pkg/generated/openapi"
	spec "k8s.io/kube-openapi/pkg/validation/spec"
)

func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {
	
	namer := openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme)

    // Convert github.com/foo/bar.Type → github.com.foo.bar.Type
    // so $ref strings use dots (no ~1 encoding) and match map keys exactly.
    getDefinitionName := func(n string) (string, spec.Extensions) {
        if strings.HasPrefix(n, "github.com/") {
            return strings.ReplaceAll(n, "/", "."), nil
        }
        return namer.GetDefinitionName(n)
    }

	mergedDefs := func(ref openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
        result := apiextensionsopenapi.GetOpenAPIDefinitions(ref)
        for k, v := range defs(ref) {
            if strings.HasPrefix(k, "github.com/") {
                result[strings.ReplaceAll(k, "/", ".")] = v
            } else {
                result[k] = v
            }
        }
        return result
    }
	
	// DEBUG: check what the namer returns for our types
    defName, _ := namer.GetDefinitionName("github.com/sdcio/config-server/apis/config/v1alpha1.Config")
    fmt.Printf("DEBUG: namer Config = %q\n", defName)
    defName2, _ := namer.GetDefinitionName("github.com/sdcio/config-server/apis/config/v1alpha1.DeviationSpec")
    fmt.Printf("DEBUG: namer DeviationSpec = %q\n", defName2)
		
	options.RecommendedConfigFns = append(options.RecommendedConfigFns, func(config *server.RecommendedConfig) *server.RecommendedConfig {
		config.OpenAPIConfig = server.DefaultOpenAPIConfig(mergedDefs, namer)
		config.OpenAPIConfig.Info.Title = name
		config.OpenAPIConfig.Info.Version = version
		config.OpenAPIConfig.GetDefinitionName = getDefinitionName

		config.OpenAPIV3Config = server.DefaultOpenAPIV3Config(mergedDefs, namer)
		config.OpenAPIV3Config.Info.Title = name
		config.OpenAPIV3Config.Info.Version = version
		config.OpenAPIConfig.GetDefinitionName = getDefinitionName
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
