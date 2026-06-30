// Command peridot is the CLI client for the peridot daemon.
package main

import (
	"fmt"
	"os"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"github.com/conallob/peridot/internal/config"
	"github.com/conallob/peridot/internal/ipc"
	"github.com/conallob/peridot/internal/mcp"
	"github.com/conallob/peridot/internal/platform"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
	// The platform implementation for the current OS is registered via the
	// build-tagged platform_<os>.go files in this package.
)

// Injected at build time via -ldflags "-X main.version=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "peridot",
		Short:   "Control the peridot wallpaper rotation daemon",
		Version: fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
	}
	root.AddCommand(
		simpleCmd("next", "Advance to the next wallpaper", func() *pb.CommandRequest {
			return &pb.CommandRequest{Command: &pb.CommandRequest_Next{Next: &pb.NextCommand{}}}
		}),
		simpleCmd("prev", "Return to the previous wallpaper", func() *pb.CommandRequest {
			return &pb.CommandRequest{Command: &pb.CommandRequest_Prev{Prev: &pb.PrevCommand{}}}
		}),
		simpleCmd("pause", "Pause automatic rotation", func() *pb.CommandRequest {
			return &pb.CommandRequest{Command: &pb.CommandRequest_Pause{Pause: &pb.PauseCommand{}}}
		}),
		simpleCmd("resume", "Resume automatic rotation", func() *pb.CommandRequest {
			return &pb.CommandRequest{Command: &pb.CommandRequest_Resume{Resume: &pb.ResumeCommand{}}}
		}),
		simpleCmd("status", "Show daemon status", func() *pb.CommandRequest {
			return &pb.CommandRequest{Command: &pb.CommandRequest_Status{Status: &pb.StatusCommand{}}}
		}),
		simpleCmd("sources", "List configured sources", func() *pb.CommandRequest {
			return &pb.CommandRequest{Command: &pb.CommandRequest_ListSources{ListSources: &pb.ListSourcesCommand{}}}
		}),
		historyCmd(),
		statsCmd(),
		fetchCmd(),
		setIntervalCmd(),
		configCmd(),
		daemonCmd(),
		mcpCmd(),
	)
	return root
}

// send dials the daemon and prints the JSON response.
func send(req *pb.CommandRequest) error {
	resp, err := ipc.NewClient("").Send(req)
	if err != nil {
		return err
	}
	printResp(resp)
	if !resp.Ok {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

func printResp(resp *pb.CommandResponse) {
	b, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp)
	if err != nil {
		fmt.Println(resp.String())
		return
	}
	fmt.Println(string(b))
}

func simpleCmd(use, short string, build func() *pb.CommandRequest) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(_ *cobra.Command, _ []string) error {
			return send(build())
		},
	}
}

func historyCmd() *cobra.Command {
	var limit int
	c := &cobra.Command{
		Use:   "history",
		Short: "Show recently displayed wallpapers",
		RunE: func(_ *cobra.Command, _ []string) error {
			return send(&pb.CommandRequest{Command: &pb.CommandRequest_History{
				History: &pb.HistoryCommand{Limit: int32(limit)},
			}})
		},
	}
	c.Flags().IntVar(&limit, "limit", 10, "maximum entries to show")
	return c
}

func fetchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch <source-id>",
		Short: "Fetch new wallpapers from a source",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return send(&pb.CommandRequest{Command: &pb.CommandRequest_FetchSource{
				FetchSource: &pb.FetchSourceCommand{SourceId: args[0]},
			}})
		},
	}
}

func setIntervalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-interval <duration>",
		Short: "Set the rotation interval (e.g. 30m, 1h)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := time.ParseDuration(args[0])
			if err != nil {
				return fmt.Errorf("invalid duration: %w", err)
			}
			return send(&pb.CommandRequest{Command: &pb.CommandRequest_SetInterval{
				SetInterval: &pb.SetIntervalCommand{Interval: durationpb.New(d)},
			}})
		},
	}
}

func configPath(cmd *cobra.Command) string {
	if p, _ := cmd.Flags().GetString("file"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return home + "/.peridot/config.toml"
}

func configCmd() *cobra.Command {
	c := &cobra.Command{Use: "config", Short: "Manage configuration"}
	c.PersistentFlags().String("file", "", "path to config.toml")

	validate := &cobra.Command{
		Use:   "validate",
		Short: "Validate the config without writing the lock file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := config.Validate(configPath(cmd)); err != nil {
				return err
			}
			fmt.Println("config is valid")
			return nil
		},
	}

	compile := &cobra.Command{
		Use:   "compile",
		Short: "Compile the config and write the .lock.pb",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Compile(configPath(cmd))
			if err != nil {
				return err
			}
			fmt.Printf("compiled: %d sources\n", len(cfg.Sources))
			return nil
		},
	}

	dump := &cobra.Command{
		Use:   "dump",
		Short: "Print the compiled config as JSON",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Validate(configPath(cmd))
			if err != nil {
				return err
			}
			b, _ := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(cfg)
			fmt.Println(string(b))
			return nil
		},
	}

	diff := &cobra.Command{
		Use:   "diff",
		Short: "Show differences between the source config and the lock file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := configPath(cmd)
			fresh, err := config.Validate(path)
			if err != nil {
				return err
			}
			locked, err := config.LoadLock(path + ".lock.pb")
			if err != nil {
				fmt.Println("no lock file; run `peridot config compile` first")
				return nil
			}
			if protoEqualJSON(fresh, locked) {
				fmt.Println("no changes")
			} else {
				fmt.Println("config differs from lock file; run `peridot config compile`")
			}
			return nil
		},
	}

	reload := &cobra.Command{
		Use:   "reload",
		Short: "Recompile the config (the daemon hot-reloads automatically)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := config.Compile(configPath(cmd)); err != nil {
				return err
			}
			fmt.Println("config recompiled; daemon will hot-reload")
			return nil
		},
	}

	c.AddCommand(validate, compile, dump, diff, reload)
	return c
}

func protoEqualJSON(a, b *pb.Config) bool {
	ab, _ := protojson.Marshal(a)
	bb, _ := protojson.Marshal(b)
	return string(ab) == string(bb)
}

func daemonCmd() *cobra.Command {
	c := &cobra.Command{Use: "daemon", Short: "Manage the daemon service"}

	withDaemon := func(short string, fn func(platform.Daemon) error) *cobra.Command {
		use := short
		return &cobra.Command{
			Use:   use,
			Short: "Daemon: " + short,
			RunE: func(_ *cobra.Command, _ []string) error {
				p := platform.Current()
				if p == nil {
					return fmt.Errorf("no platform support for this OS")
				}
				return fn(p.Daemon())
			},
		}
	}

	c.AddCommand(
		withDaemon("start", func(d platform.Daemon) error { return d.Start() }),
		withDaemon("stop", func(d platform.Daemon) error { return d.Stop() }),
		withDaemon("restart", func(d platform.Daemon) error { return d.Restart() }),
		withDaemon("install", func(d platform.Daemon) error { return d.Install() }),
		&cobra.Command{
			Use:   "status",
			Short: "Daemon: status",
			RunE: func(_ *cobra.Command, _ []string) error {
				p := platform.Current()
				if p == nil {
					return fmt.Errorf("no platform support for this OS")
				}
				s, err := p.Daemon().Status()
				if err != nil {
					return err
				}
				fmt.Println(s)
				return nil
			},
		},
	)
	return c
}

func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run the MCP stdio server bridging to the daemon",
		RunE: func(_ *cobra.Command, _ []string) error {
			return mcp.Serve(ipc.DefaultSocketPath())
		},
	}
}
