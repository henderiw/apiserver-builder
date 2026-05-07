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
	spec "k8s.io/kube-openapi/pkg/validation/spec"
)

// goImportPathToReverseDNS converts any Go import path to a stable reverse-DNS
// component name — the same convention used by core Kubernetes types (io.k8s.*).
//
//   github.com/sdcio/config-server/apis/config/v1alpha1.ConfigStatus
//   → com.github.sdcio.config-server.apis.config.v1alpha1.ConfigStatus
//
//   k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta
//   → io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta
//
// The result contains no "/" or "~", so kube-openapi's EscapeJsonPointer is a no-op
// and $ref values produced from it are always stable (no double-encoding issues).
func goImportPathToReverseDNS(goPath string) string {
	slash := strings.Index(goPath, "/")
	if slash == -1 {
		return goPath
	}
	host := goPath[:slash]
	rest := strings.ReplaceAll(goPath[slash+1:], "/", ".")
	parts := strings.Split(host, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ".") + "." + rest
}

// WithOpenAPIDefinitions registers OpenAPI definitions for the API server.
//
// # The two-spec problem
//
// There are two distinct OpenAPI specs in play with very different consumers:
//
//  1. The SERVED spec  (/openapi/v2, /openapi/v3) — consumed by kubectl's
//     client-side validator. kube-openapi calls EscapeJsonPointer() on the
//     output of GetDefinitionName when building $ref strings. If GetDefinitionName
//     returns a name that already contains "~" (e.g. the namer's fallback for
//     non-GVK types: "github.com~1sdcio~1…TargetStatus"), EscapeJsonPointer
//     double-encodes the "~" → "~0", yielding "~01". kubectl resolves $ref
//     tokens literally (no JSON Pointer decode) so "~01" ≠ "~1" → not found.
//     Fix: use our getDefinitionName wrapper which converts all Go import-path
//     keys to clean reverse-DNS names (no "~" or "/") before EscapeJsonPointer
//     sees them.
//
//  2. The FIELD-MANAGER spec — built by buildOpenAPIModels inside the Kubernetes
//     generic apiserver using the namer's GetDefinitionName DIRECTLY (it does not
//     read config.OpenAPIV3Config.GetDefinitionName). For non-GVK types the namer
//     returns the name unchanged (e.g. "github.com/…/TargetStatus"), then
//     EscapeJsonPointer turns "/" → "~1", making $ref targets like
//     "github.com~1sdcio~1…TargetStatus". The TypeConverter strips the
//     "#/components/schemas/" prefix and does a direct map lookup for that string.
//     The definitions map only has "/" keyed entries, so the lookup fails with
//     "no type found matching: github.com~1…TargetStatus".
//     Fix: add "~1"-encoded alias entries to mergedDefs so the field-manager's
//     spec builder finds them when following those $refs.
//
// With both fixes applied:
//   - Served spec:  $ref uses clean reverse-DNS → kubectl resolves correctly.
//   - FM spec:      $ref uses ~1 names (namer path) → alias entries are found.
func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {

	namer := openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme)

	// getDefinitionName is used by the SERVED spec only.
	// It delegates to the namer for GVK-registered types (preserving their
	// x-kubernetes-group-version-kind extensions). For everything else the namer
	// returns the name unchanged — which contains "/" and would trigger double-
	// encoding via EscapeJsonPointer — so we replace the fallback with a proper
	// reverse-DNS transformation.
	getDefinitionName := func(n string) (string, spec.Extensions) {
		resolved, ext := namer.GetDefinitionName(n)
		// Namer fallback returns the original name unchanged; it contains "/" for
		// Go import paths. Reverse-DNS names never contain "/" or "~".
		if strings.ContainsAny(resolved, "/~") {
			return goImportPathToReverseDNS(n), nil
		}
		return resolved, ext
	}

	// mergedDefs merges apiextensions built-ins with the user's definitions and
	// adds "~1"-encoded aliases for every "github.com/" type.  The aliases are
	// consumed exclusively by the field-manager spec builder (which uses the namer
	// path and therefore produces "~1"-encoded $refs as lookup keys).  The served
	// spec builder uses getDefinitionName above and never references these aliases
	// (its $refs are reverse-DNS), so they remain invisible to kubectl.
	mergedDefs := func(ref openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
		result := apiextensionsopenapi.GetOpenAPIDefinitions(ref)
		for k, v := range defs(ref) {
			result[k] = v
		}
		// Add ~1-encoded aliases so that the field-manager's spec builder can
		// resolve $refs of the form "github.com~1sdcio~1…TargetStatus" back to
		// the actual schema definition.
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
		config.OpenAPIConfig.GetDefinitionName = getDefinitionName // served v2 spec

		config.OpenAPIV3Config = server.DefaultOpenAPIV3Config(mergedDefs, namer)
		config.OpenAPIV3Config.Info.Title = name
		config.OpenAPIV3Config.Info.Version = version
		config.OpenAPIV3Config.GetDefinitionName = getDefinitionName // served v3 spec

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