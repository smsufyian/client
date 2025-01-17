// Copyright © 2019 The Knative Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ping

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"knative.dev/client/pkg/kn/commands"
	"knative.dev/client/pkg/kn/commands/flags"
	sourcesv1beta2 "knative.dev/client/pkg/sources/v1beta2"
	"knative.dev/client/pkg/util"
)

// NewPingUpdateCommand prepares the command for a PingSource update
func NewPingUpdateCommand(p *commands.KnParams) *cobra.Command {
	var updateFlags pingUpdateFlags
	var sinkFlags flags.SinkFlags

	cmd := &cobra.Command{
		Use:   "update NAME",
		Short: "Update a ping source",
		Example: `
  # Update the schedule of a Ping source 'my-ping' to fire every minute
  kn source ping update my-ping --schedule "* * * * *"`,

		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) != 1 {
				return errors.New("name of Ping source required")
			}
			name := args[0]

			pingSourceClient, err := newPingSourceClient(p, cmd)
			if err != nil {
				return err
			}

			namespace, err := p.GetNamespace(cmd)
			if err != nil {
				return err
			}
			dynamicClient, err := p.NewDynamicClient(namespace)
			if err != nil {
				return err
			}

			source, err := pingSourceClient.GetPingSource(cmd.Context(), name)
			if err != nil {
				return err
			}
			if source.GetDeletionTimestamp() != nil {
				return fmt.Errorf("can't update ping source %s because it has been marked for deletion", name)
			}

			b := sourcesv1beta2.NewPingSourceBuilderFromExisting(source)
			if cmd.Flags().Changed("schedule") {
				b.Schedule(updateFlags.schedule)
			}

			data, dataBase64, err := getDataFields(&updateFlags)
			if err != nil {
				return fmt.Errorf("cannot update PingSource %q in namespace "+
					"%q because: %s", name, namespace, err)
			}

			if cmd.Flags().Changed("data") {
				b.Data(data).DataBase64(dataBase64)
			}
			if cmd.Flags().Changed("sink") {
				destination, err := sinkFlags.ResolveSink(cmd.Context(), dynamicClient, namespace)
				if err != nil {
					return err
				}
				b.Sink(*destination)
			}

			if cmd.Flags().Changed("ce-override") {
				ceOverridesMap, err := util.MapFromArrayAllowingSingles(updateFlags.ceOverrides, "=")
				if err != nil {
					return err
				}
				ceOverridesToRemove := util.ParseMinusSuffix(ceOverridesMap)
				b.CloudEventOverrides(ceOverridesMap, ceOverridesToRemove)
			}

			err = pingSourceClient.UpdatePingSource(cmd.Context(), b.Build())
			if err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Ping source '%s' updated in namespace '%s'.\n", name, pingSourceClient.Namespace())
			}
			return err
		},
	}
	commands.AddNamespaceFlags(cmd.Flags(), false)
	updateFlags.addFlags(cmd)
	sinkFlags.Add(cmd)

	return cmd
}
