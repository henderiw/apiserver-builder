package builder

import (
	"strings"

	"github.com/henderiw/apiserver-builder/pkg/apiserver"
	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder/options"
	apiextensionsopenapi "k8s.io/apiextensions-apiserver/pkg/generated/openapi"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	scheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	openapicommon "k8s.io/kube-openapi/pkg/common"
	openapiutil "k8s.io/kube-openapi/pkg/util"
	spec "k8s.io/kube-openapi/pkg/validation/spec"
)

// WithOpenAPIDefinitions registers OpenAPI definitions for the API server.
//
// Definition names are reverse-DNS and slash-free, e.g.
//
//	github.com/sdcio/config-server/apis/config/v1alpha1.ConfigSpec
//	  -> com.github.sdcio.config-server.apis.config.v1alpha1.ConfigSpec
//
// Because the names contain neither "/" nor "~", JSON-pointer escaping is a
// no-op, so:
//   - $ref == component key (raw lookups in the TypeConverter and
//     BuildOpenAPIDefinitionsForResources work)
//   - the refs are valid per RFC 6901 (standards-compliant clients resolve them)
//   - registered kinds get x-kubernetes-group-version-kind from the namer
//
// We still use our own ref callback and add alias entries keyed by the
// friendly name, so that resolution doesn't depend on kube-openapi internals.
func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {

	namer := openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme)

	// Delegate to the namer (gives friendly name + GVK extension for registered
	// kinds); guarantee a slash-free name even if the namer passes it through.
	getDefinitionName := func(n string) (string, spec.Extensions) {
		defName, ext := namer.GetDefinitionName(n)
		if strings.ContainsAny(defName, "/~") {
			defName = openapiutil.ToRESTFriendlyName(n)
		}
		return defName, ext
	}

	makeGetDefinitions := func(refPrefix string) openapicommon.GetOpenAPIDefinitions {
		return func(_ openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
			ref := func(typeName string) spec.Ref {
				defName, _ := getDefinitionName(typeName)
				return spec.MustCreateRef(refPrefix + defName)
			}

			result := apiextensionsopenapi.GetOpenAPIDefinitions(ref)
			for k, v := range defs(ref) {
				result[k] = v
			}

			// Alias every Go-path key under its friendly name so lookups by
			// either form succeed.
			aliases := make(map[string]openapicommon.OpenAPIDefinition, len(result))
			for k, v := range result {
				if friendly, _ := getDefinitionName(k); friendly != k {
					aliases[friendly] = v
				}
			}
			for k, v := range aliases {
				result[k] = v
			}

			klog.V(4).Infof("openapi defs (%s): %d definitions, %d aliases", refPrefix, len(result), len(aliases))
			return result
		}
	}

	options.RecommendedConfigFns = append(options.RecommendedConfigFns, func(config *server.RecommendedConfig) *server.RecommendedConfig {
		config.OpenAPIConfig = server.DefaultOpenAPIConfig(makeGetDefinitions("#/definitions/"), namer)
		config.OpenAPIConfig.Info.Title = name
		config.OpenAPIConfig.Info.Version = version
		config.OpenAPIConfig.GetDefinitionName = getDefinitionName

		config.OpenAPIV3Config = server.DefaultOpenAPIV3Config(makeGetDefinitions("#/components/schemas/"), namer)
		config.OpenAPIV3Config.Info.Title = name
		config.OpenAPIV3Config.Info.Version = version
		config.OpenAPIV3Config.GetDefinitionName = getDefinitionName

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