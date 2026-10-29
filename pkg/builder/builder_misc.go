package builder

import (
	"strings"
	"sync"

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

func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {

	// Lazily build the namer on first use. NewDefinitionNamer snapshots the
	// scheme, so it must run after all resources are registered. First use is
	// when the API groups are installed / the spec is built, which is late enough.
	var (
		once  sync.Once
		namer *openapinamer.DefinitionNamer
	)
	getNamer := func() *openapinamer.DefinitionNamer {
		once.Do(func() {
			for gvk, t := range apiserver.Scheme.AllKnownTypes() {
				if gvk.Group == "config.sdcio.dev" || gvk.Group == "inv.sdcio.dev" {
					klog.Infof("scheme: %s -> %s.%s", gvk.String(), t.PkgPath(), t.Name())
				}
			}
			namer = openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme)
			_, ext := namer.GetDefinitionName("github.com/sdcio/config-server/apis/config/v1alpha1.ConfigSet")
			klog.Infof("openapi namer built: %d known types, ConfigSet GVK ext=%v",
				len(apiserver.Scheme.AllKnownTypes()), ext)
		})
		return namer
	}

	getDefinitionName := func(n string) (string, spec.Extensions) {
		defName, ext := getNamer().GetDefinitionName(n)
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

	options.RecommendedConfigFns = append(options.RecommendedConfigFns, func(config *server.RecommendedConfig) *server.RecommendedConfig {
		// DefaultOpenAPI*Config wants a namer for its default; we override
		// GetDefinitionName right after, so pass a placeholder built from an
		// empty scheme set — it is never used.
		placeholder := openapinamer.NewDefinitionNamer()

		config.OpenAPIConfig = server.DefaultOpenAPIConfig(makeGetDefinitions("#/definitions/"), placeholder)
		config.OpenAPIConfig.Info.Title = name
		config.OpenAPIConfig.Info.Version = version
		config.OpenAPIConfig.GetDefinitionName = getDefinitionName

		config.OpenAPIV3Config = server.DefaultOpenAPIV3Config(makeGetDefinitions("#/components/schemas/"), placeholder)
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
