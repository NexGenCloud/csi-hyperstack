package main

import (
	"os"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/component-base/cli"
	"k8s.io/csi-hyperstack/pkg/driver"
	"k8s.io/klog/v2"

	"context"
)

var (
	name    string
	version string
)

func main() {
	viper.SetEnvPrefix("HYPERSTACK")
	viper.AutomaticEnv()

	_ = viper.BindEnv("hyperstack-api-key", "HYPERSTACK_API_KEY")
	_ = viper.BindEnv("hyperstack-api-address", "HYPERSTACK_API_ADDRESS")
	// _ = viper.BindEnv("hyperstack-environment", "HYPERSTACK_ENVIRONMENT")
	viper.SetDefault("endpoint", "unix://var/run/csi.sock")
	viper.SetDefault("metrics-enabled", true)
	viper.SetDefault("http-endpoint", ":8080")

	rootCmd := &cobra.Command{
		Use:   name,
		Short: "CSI based Hyperstack driver",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		Version: version,
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the CSI based Hyperstack driver",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := driverStart(cmd.Context())
			if err != nil {
				klog.Errorf("Failed to start driver: %v", err)
				return err
			}
			return nil
		},
	}

	flags := startCmd.PersistentFlags()
	flags.SortFlags = false
	flags.String("endpoint", viper.GetString("endpoint"), "CSI gRPC endpoint")
	flags.Bool("metrics-enabled", viper.GetBool("metrics-enabled"), "Enables metrics endpoint")
	flags.String("http-endpoint", viper.GetString("http-endpoint"), "HTTP endpoint")
	// flags.String("hyperstack-cluster-id", "", "Hyperstack cluster identifier")
	// flags.String("hyperstack-node-id", "", "Hyperstack node identifier")
	flags.String("hyperstack-api-key", viper.GetString("hyperstack-api-key"), "Hyperstack API key (env: HYPERSTACK_API_KEY)")
	flags.String("hyperstack-api-address", viper.GetString("hyperstack-api-address"), "Hyperstack API server address (env: HYPERSTACK_API_ADDRESS)")
	// flags.String("hyperstack-environment", viper.GetString("hyperstack-environment"), "Hyperstack environment name")
	flags.Bool("service-controller-enabled", false, "Enables CSI controller service")
	flags.Bool("service-node-enabled", false, "Enables CSI node service")

	// _ = startCmd.MarkFlagRequired("hyperstack-cluster-id")
	// _ = startCmd.MarkFlagRequired("hyperstack-node-id")
	_ = startCmd.MarkFlagRequired("hyperstack-api-key")
	_ = startCmd.MarkFlagRequired("hyperstack-api-address")
	// _ = startCmd.MarkFlagRequired("hyperstack-environment")

	rootCmd.AddCommand(startCmd)

	cobra.OnInitialize(func() {
		err := viper.BindPFlags(flags)
		if err != nil {
			klog.Errorf("%v", err)
		}
	})

	rootCmd.SetHelpTemplate(helpTemplate())

	if len(os.Args) < 2 {
		_ = rootCmd.Help()
		os.Exit(0)
	}

	code := cli.Run(rootCmd)
	os.Exit(code)
}

func helpTemplate() string {
	return `{{with (or .Long .Short)}}{{.}}{{end}}

Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (and .IsAvailableCommand (not .IsAdditionalHelpTopicCommand))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

Environment variables:
  HYPERSTACK_API_KEY                 Hyperstack API key
  HYPERSTACK_API_ADDRESS             Hyperstack API server address

Use "{{.CommandPath}} [command] --help" for more information about a command.
`
}

func driverStart(ctx context.Context) (err error) {
	drv := driver.NewDriver(&driver.DriverOpts{
		Endpoint: viper.GetString("endpoint"),
		// HyperstackClusterId:  viper.GetString("hyperstack-cluster-id"),
		// HyperstackNodeId:     viper.GetString("hyperstack-node-id"),
		HyperstackApiKey:     viper.GetString("hyperstack-api-key"),
		HyperstackApiAddress: viper.GetString("hyperstack-api-address"),
		// Environment:          viper.GetString("hyperstack-environment"),
	})

	drv.SetupIdentityService()

	if viper.GetBool("service-controller-enabled") {
		drv.SetupControllerService()
	}

	if viper.GetBool("service-node-enabled") {
		drv.SetupNodeService()
	}

	wg := &sync.WaitGroup{}

	// Waiting for anything to stop
	wg.Add(1)

	srvHttp := driver.RunHttpServer(
		ctx,
		wg,
		viper.GetString("http-endpoint"),
		viper.GetBool("metrics-enabled"),
	)

	srvGRPC, err := drv.Run(ctx, wg)
	if err != nil {
		return err
	}

	wg.Wait()

	klog.Info("Stopping gRPC server")
	srvGRPC.Stop()

	klog.Info("Stopping HTTP server")
	err = srvHttp.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}
