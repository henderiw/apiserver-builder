package options

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/henderiw/apiserver-builder/pkg/apiserver"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
)

type ServerOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions

	StdOut io.Writer
	StdErr io.Writer
}

// NewServerOptions returns a new ServerOptions
func NewServerOptions(out, errOut io.Writer, versions ...schema.GroupVersion) *ServerOptions {
	apiserver.Codecs = serializer.NewCodecFactory(apiserver.Scheme)

	o := &ServerOptions{
		RecommendedOptions: genericoptions.NewRecommendedOptions(
			EtcdPath,
			apiserver.Codecs.LegacyCodec(versions...),
		),

		StdOut: out,
		StdErr: errOut,
	}
	o.RecommendedOptions.Etcd.StorageConfig.EncodeVersioner = schema.GroupVersions(versions)
	return o
}

// Complete fills in fields required to have valid data
func (o *ServerOptions) Complete() error {
	ApplyServerOptionsFns(o)
	return nil
}

// Validate validates ServerOptions
func (o ServerOptions) Validate(args []string) error {
	errors := []error{}
	errors = append(errors, o.RecommendedOptions.Validate()...)
	return utilerrors.NewAggregate(errors)
}

// RunServer starts a new Server given ServerOptions
func (o ServerOptions) RunServer(ctx context.Context, serverName string) error {
	config, err := o.Config(serverName)
	if err != nil {
		return err
	}

	server, err := config.Complete().New(ctx)
	if err != nil {
		return err
	}

	server.GenericAPIServer.AddPostStartHookOrDie(fmt.Sprintf("start-%s-informers", serverName), func(context genericapiserver.PostStartHookContext) error {
		if config.GenericConfig.SharedInformerFactory != nil {
			config.GenericConfig.SharedInformerFactory.Start(context.StopCh)
		}
		return nil
	})

	return server.GenericAPIServer.PrepareRun().Run(ctx.Done())
}

// Config returns config for the api server given WardleServerOptions
func (o *ServerOptions) Config(serverName string) (*apiserver.Config, error) {
	// TODO have a "real" external address
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	// change: allow etcd options to be nil
	// TODO: this should be reverted after rebasing sample-apiserver onto https://github.com/kubernetes/kubernetes/pull/101106
	//if o.RecommendedOptions.Etcd != nil {
	//o.RecommendedOptions.Etcd.StorageConfig.Paging = utilfeature.DefaultFeatureGate.Enabled(features.APIListChunking)
	//}

	serverConfig := genericapiserver.NewRecommendedConfig(apiserver.Codecs)

	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	serverConfig = ApplyRecommendedConfigFns(serverConfig)

	config := &apiserver.Config{
		GenericConfig: serverConfig,
		ExtraConfig: apiserver.ExtraConfig{
			ServerName: serverName,
		},
	}
	return config, nil
}
