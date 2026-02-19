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
	spec "k8s.io/kube-openapi/pkg/validation/spec"
)

/*
Root cause: NewTypeConverter builds an SMD type universe from StaticOpenAPISpec. When it resolves a $ref like #/components/schemas/github.com~1sdcio~1config-server~1apis~1config~1v1alpha1.DeviationSpec, it strips the prefix and does a direct map lookup for github.com~1sdcio~1...DeviationSpec. But GetOpenAPIDefinitions returns keys with raw / (github.com/sdcio/...). The mismatch meant DeviationSpec and DeviationStatus were never found.
Fix (two parts in builder_misc.go):

GetDefinitionName override — returns ~1-encoded names for github.com/ types so that $ref strings and map keys are consistent
mergedDefs alias loop — adds ~1-encoded duplicate entries alongside the original / keys so that both getResourceNamesForGroup (uses /) and NewTypeConverter (uses ~1) can find them
*/

func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {
	
	namer := openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme)

    // Return ~1-encoded names for github.com/ types so StaticOpenAPISpec
    // keys match what NewTypeConverter looks up when resolving $refs.
    getDefinitionName := func(n string) (string, spec.Extensions) {
        if strings.HasPrefix(n, "github.com/") {
            return strings.ReplaceAll(n, "/", "~1"), nil
        }
        return namer.GetDefinitionName(n)
    }

	mergedDefs := func(ref openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
        result := apiextensionsopenapi.GetOpenAPIDefinitions(ref)
        for k, v := range defs(ref) {
            result[k] = v
        }
        // Add ~1-encoded aliases alongside original keys.
        // getResourceNamesForGroup looks up by / keys, BuildOpenAPIDefinitionsForResources
        // may look up by ~1 keys (via GetDefinitionName). Provide both.
        aliases := make(map[string]openapicommon.OpenAPIDefinition)
        for k, v := range result {
            if strings.HasPrefix(k, "github.com/") {
                aliases[strings.ReplaceAll(k, "/", "~1")] = v
            }
        }
        for k, v := range aliases {
            result[k] = v
        }
        return result
    }
	
	options.RecommendedConfigFns = append(options.RecommendedConfigFns, func(config *server.RecommendedConfig) *server.RecommendedConfig {
        config.OpenAPIConfig = server.DefaultOpenAPIConfig(mergedDefs, namer)
        config.OpenAPIConfig.Info.Title = name
        config.OpenAPIConfig.Info.Version = version
        config.OpenAPIConfig.GetDefinitionName = getDefinitionName  // V2

        config.OpenAPIV3Config = server.DefaultOpenAPIV3Config(mergedDefs, namer)
        config.OpenAPIV3Config.Info.Title = name
        config.OpenAPIV3Config.Info.Version = version
        config.OpenAPIV3Config.GetDefinitionName = getDefinitionName  // V3 - was missing!
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
