package builder

import (
	"context"
	"flag"
	"os"

	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
)

// APIServer builds an apiserver to server Kubernetes resources and sub resources.
var APIServer = &Server{
	storageProvider: map[schema.GroupResource]*singletonProvider{},
}

// Server builds a new apiserver for a single API group
type Server struct {
	ServerName           string
	EtcdPath             string
	errs                 []error
	storageProvider      map[schema.GroupResource]*singletonProvider
	groupVersions        map[schema.GroupVersion]bool
	orderedGroupVersions []schema.GroupVersion
	schemes              []*runtime.Scheme
	schemeBuilder        runtime.SchemeBuilder
}

// Build returns a Command used to run the apiserver
func (r *Server) Build(ctx context.Context) (*Command, error) {
	apiserverbuilder.EtcdPath = r.EtcdPath
	r.schemes = append(r.schemes, apiserverbuilder.Scheme)
	r.schemeBuilder.Register(
		func(scheme *runtime.Scheme) error {
			groupVersions := make(map[string]sets.Set[string])
			for gvr := range apiserverbuilder.APIs {
				if groupVersions[gvr.Group] == nil {
					groupVersions[gvr.Group] = sets.New[string]()
				}
				groupVersions[gvr.Group].Insert(gvr.Version)
			}
			for g, versions := range groupVersions {
				gvs := []schema.GroupVersion{}
				for v := range versions {
					gvs = append(gvs, schema.GroupVersion{
						Group:   g,
						Version: v,
					})
				}
				err := scheme.SetVersionPriority(gvs...)
				if err != nil {
					return err
				}
			}
			for i := range r.orderedGroupVersions {
				metav1.AddToGroupVersion(scheme, r.orderedGroupVersions[i])
			}
			return nil
		},
	)
	for i := range r.schemes {
		if err := r.schemeBuilder.AddToScheme(r.schemes[i]); err != nil {
			panic(err)
		}
	}

	if len(r.errs) != 0 {
		return nil, errs{list: r.errs}
	}
	o := apiserverbuilder.NewServerOptions(os.Stdout, os.Stderr, r.orderedGroupVersions...)
	cmd := apiserverbuilder.NewCommandStartServer(ctx, r.ServerName, o)
	apiserverbuilder.ApplyFlagsFns(cmd.Flags())
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	return cmd, nil
}

// Execute builds and executes the apiserver Command.
func (r *Server) Execute(ctx context.Context) error {
	cmd, err := r.Build(ctx)
	if err != nil {
		return err
	}
	return cmd.Execute()
}
