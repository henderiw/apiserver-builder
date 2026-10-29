package builder

import (
	"sort"
	"sync"

	"github.com/henderiw/apiserver-builder/pkg/apiserver"
	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder/options"
	apiextensionsopenapi "k8s.io/apiextensions-apiserver/pkg/generated/openapi"
	"k8s.io/apimachinery/pkg/runtime"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/klog/v2"
	openapicommon "k8s.io/kube-openapi/pkg/common"
	openapiutil "k8s.io/kube-openapi/pkg/util"
	spec "k8s.io/kube-openapi/pkg/validation/spec"
)

const extensionGVK = "x-kubernetes-group-version-kind"

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
//
// x-kubernetes-group-version-kind is built from apiserver.Scheme on first use
// (after all resources are registered) for every non-internal registered type.
// Server-side apply's TypeConverter uses it to map GVK -> schema. We do this
// ourselves because openapinamer.DefinitionNamer returned no GVKs for our types.
//
// We use our own ref callback and add alias entries keyed by the friendly
// name, so that resolution doesn't depend on kube-openapi internals.
func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {

	// Go type path -> GVK extension, built lazily from apiserver.Scheme
	// (after all resources are registered).
	var (
		once     sync.Once
		gvkByTyp map[string][]interface{}
	)
	getGVKs := func() map[string][]interface{} {
		once.Do(func() {
			type gvkEntry struct{ g, v, k string }
			tmp := map[string][]gvkEntry{}
			for gvk, t := range apiserver.Scheme.AllKnownTypes() {
				if gvk.Version == runtime.APIVersionInternal {
					continue
				}
				key := t.PkgPath() + "." + t.Name()
				tmp[key] = append(tmp[key], gvkEntry{gvk.Group, gvk.Version, gvk.Kind})
			}
			gvkByTyp = make(map[string][]interface{}, len(tmp))
			for key, entries := range tmp {
				sort.Slice(entries, func(i, j int) bool {
					if entries[i].g != entries[j].g {
						return entries[i].g < entries[j].g
					}
					if entries[i].v != entries[j].v {
						return entries[i].v < entries[j].v
					}
					return entries[i].k < entries[j].k
				})
				list := make([]interface{}, 0, len(entries))
				for _, e := range entries {
					list = append(list, map[string]interface{}{"group": e.g, "version": e.v, "kind": e.k})
				}
				gvkByTyp[key] = list
			}
			klog.V(2).Infof("openapi GVK map built: %d types", len(gvkByTyp))
		})
		return gvkByTyp
	}

	getDefinitionName := func(n string) (string, spec.Extensions) {
		defName := openapiutil.ToRESTFriendlyName(n)
		if gvks, ok := getGVKs()[n]; ok {
			return defName, spec.Extensions{extensionGVK: gvks}
		}
		return defName, nil
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
		// DefaultOpenAPI*Config needs a namer; we override GetDefinitionName right after.
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