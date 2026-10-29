package builder

import (
	"strings"

	"github.com/henderiw/apiserver-builder/pkg/apiserver"
	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder/options"
	apiextensionsopenapi "k8s.io/apiextensions-apiserver/pkg/generated/openapi"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	scheme "k8s.io/client-go/kubernetes/scheme"
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
// so JSON-pointer escaping is a no-op: $ref == component key, refs are valid
// per RFC 6901, and the TypeConverter's raw lookups work.
//
// The namer is created inside the RecommendedConfigFn (at server start), after
// apiserver.Scheme is fully populated, so every registered kind gets its
// x-kubernetes-group-version-kind extension from the namer. That extension is
// what server-side apply's TypeConverter uses to map GVK -> schema.
func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {

	options.RecommendedConfigFns = append(options.RecommendedConfigFns, func(config *server.RecommendedConfig) *server.RecommendedConfig {
		// Build the namer now: apiserver.Scheme is fully populated at this point,
		// so registered kinds get x-kubernetes-group-version-kind.
		namer := openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme)

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
				aliases := make(map[string]openapicommon.OpenAPIDefinition, len(result))
				for k, v := range result {
					if friendly, _ := getDefinitionName(k); friendly != k {
						aliases[friendly] = v
					}
				}
				for k, v := range aliases {
					result[k] = v
				}
				return result
			}
		}

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
