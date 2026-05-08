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
	spec "k8s.io/kube-openapi/pkg/validation/spec"
)

// WithOpenAPIDefinitions registers OpenAPI definitions for the API server.
//
// # Root cause of the ~1 / SSA / kubectl issue
//
// kube-openapi builds $ref strings as:
//
//	"#/.../schemas/" + EscapeJsonPointer(GetDefinitionName(goPath))
//
// GetDefinitionName for non-GVK types falls back to returning the name
// unchanged (containing "/").  EscapeJsonPointer then converts "/" → "~1".
//
// If we override GetDefinitionName to return "~1"-encoded names (which is what
// we need so that component KEYS are "~1"-encoded and the TypeConverter can
// find them), EscapeJsonPointer double-encodes the "~" → "~0", yielding "~01"
// in the $ref.  In newer kube-openapi, BuildOpenAPIDefinitionsForResources
// finds sub-types by looking up the raw $ref content in the definitions map.
// "~01" is NOT a key in the map (map has "~1" aliases and "/" Go-path keys),
// so TargetStatus/ConfigStatus/etc. are never included in the TypeConverter's
// schema.  The TypeConverter then fails with "no type found matching: ~1-name".
//
// # Fix: bypass EscapeJsonPointer with a custom ReferenceCallback
//
// By providing our OWN ref callback to GetOpenAPIDefinitions — one that inserts
// the GetDefinitionName output DIRECTLY into the $ref without calling
// EscapeJsonPointer — we ensure:
//
//   $ref  = GetDefinitionName(goPath)  = "github.com~1sdcio~1...TargetStatus"
//   key   = GetDefinitionName(goPath)  = "github.com~1sdcio~1...TargetStatus"
//   $ref == key  →  raw lookup always works, no double-encoding
//
// This satisfies BOTH the TypeConverter (which does a raw lookup of the $ref
// content in its schema) AND kubectl (which looks up component names literally).
//
// The "~1"-encoded alias entries in the definitions map are still needed so
// that BuildOpenAPIDefinitionsForResources can find root types when
// getResourceNamesForGroup passes their canonical (~1-encoded) names.
func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {

	klog.Infof("apiserver-builder/WithOpenAPIDefinitions: %q %q (myRef-no-escape build)", name, version)

	namer := openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme)

	// getDefinitionName: returns ~1-encoded names for github.com/ types so that
	// component keys == $ref tokens (both ~1-encoded, no EscapeJsonPointer needed).
	// For all other types (io.k8s.*, etc.) we delegate to the namer which
	// returns proper names (reverse-DNS for registered types, unchanged for others).
	getDefinitionName := func(n string) (string, spec.Extensions) {
		if strings.HasPrefix(n, "github.com/") {
			return strings.ReplaceAll(n, "/", "~1"), nil
		}
		return namer.GetDefinitionName(n)
	}

	// makeGetDefinitions returns a GetOpenAPIDefinitions function that:
	//   1. Uses a CUSTOM ref callback (myRef) that inserts getDefinitionName output
	//      directly into $ref WITHOUT calling EscapeJsonPointer.
	//   2. Merges apiextensions built-ins with the user's definitions.
	//   3. Adds "~1"-encoded alias entries alongside the original Go-path keys so
	//      that BuildOpenAPIDefinitionsForResources can look up canonical
	//      (~1-encoded) names returned by getResourceNamesForGroup.
	makeGetDefinitions := func(refPrefix string) openapicommon.GetOpenAPIDefinitions {
		return func(_ openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
			// myRef produces $refs like "#/components/schemas/github.com~1sdcio~1...X"
			// without any EscapeJsonPointer call, so $ref == component key.
			myRef := func(typeName string) spec.Ref {
				defName, _ := getDefinitionName(typeName)
				return spec.MustCreateRef(refPrefix + defName)
			}

			result := apiextensionsopenapi.GetOpenAPIDefinitions(myRef)
			for k, v := range defs(myRef) {
				result[k] = v
			}

			// Add ~1-encoded alias entries.  getResourceNamesForGroup calls
			// config.GetDefinitionName on each registered Go type to get canonical
			// names; for github.com/ types those are ~1-encoded.  The spec builder
			// then looks these up in the definitions map.  Without aliases the
			// lookup fails and the types are excluded from the TypeConverter schema.
			aliases := make(map[string]openapicommon.OpenAPIDefinition)
			for k, v := range result {
				if strings.HasPrefix(k, "github.com/") {
					aliases[strings.ReplaceAll(k, "/", "~1")] = v
				}
			}
			for k, v := range aliases {
				result[k] = v
			}

			klog.V(4).Infof("makeGetDefinitions(%s): %d definitions (%d ~1-encoded aliases)",
				refPrefix, len(result), len(aliases))
			return result
		}
	}

	options.RecommendedConfigFns = append(options.RecommendedConfigFns, func(config *server.RecommendedConfig) *server.RecommendedConfig {
		klog.Infof("apiserver-builder: applying OpenAPI config (myRef-no-escape)")

		// v2 config — used by getOpenAPIModels → BuildOpenAPIDefinitionsForResources
		// → TypeConverter.  myRef gives it ~1-encoded $refs that match the ~1-encoded
		// component keys produced by getDefinitionName.
		config.OpenAPIConfig = server.DefaultOpenAPIConfig(makeGetDefinitions("#/definitions/"), namer)
		config.OpenAPIConfig.Info.Title = name
		config.OpenAPIConfig.Info.Version = version
		config.OpenAPIConfig.GetDefinitionName = getDefinitionName

		// v3 config — served to kubectl.  Same myRef approach; $refs are ~1-encoded
		// (not double-encoded ~01) so kubectl's raw component lookup succeeds.
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