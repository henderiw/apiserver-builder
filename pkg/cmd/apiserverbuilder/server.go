package apiserverbuilder

import (
	"context"
	"fmt"

	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder/options"
	"github.com/spf13/cobra"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
)

// NewCommandStartServer provides a CLI handler for 'start master' command
// with a default ServerOptions.
func NewCommandStartServer(ctx context.Context, serverName string, defaults *options.ServerOptions) *cobra.Command {
	o := *defaults
	cmd := &cobra.Command{
		Short: fmt.Sprintf("launch %s", serverName),
		Long:  fmt.Sprintf("launch %s", serverName),
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.RunServer(ctx, serverName); err != nil {
				return err
			}
			return nil
		},
	}

	flags := cmd.Flags()
	o.RecommendedOptions.AddFlags(flags)
	utilfeature.DefaultMutableFeatureGate.AddFlag(flags)

	return cmd
}
