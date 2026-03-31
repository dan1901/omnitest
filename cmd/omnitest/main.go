package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/omnitest/omnitest/internal/agent"
	"github.com/omnitest/omnitest/internal/api"
	"github.com/omnitest/omnitest/internal/config"
	"github.com/omnitest/omnitest/internal/controller"
	"github.com/omnitest/omnitest/internal/output"
	"github.com/omnitest/omnitest/internal/report"
	"github.com/omnitest/omnitest/internal/runner"
	"github.com/omnitest/omnitest/internal/threshold"
)

var version = "dev"

// Exit codes
const (
	exitOK            = 0
	exitThresholdFail = 1
	exitScenarioError = 2
	exitConnectionErr = 3
	exitInternalError = 99
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omnitest",
		Short: "OmniTest - Cloud-native performance testing tool",
		Long:  "OmniTest is a YAML-driven HTTP load testing tool with real-time metrics and threshold-based pass/fail.",
	}

	// Global flags
	var noColor bool
	var verbose bool
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose/debug output")

	// run command
	var (
		vusersOverride   int
		durationOverride string
		rampUpOverride   string
		outputFormats    []string
		outDir           string
		quiet            bool
		envVars          []string
	)

	runCmd := &cobra.Command{
		Use:   "run <scenario.yaml>",
		Short: "Run a load test from a YAML scenario file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 환경변수 설정
			for _, env := range envVars {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					os.Setenv(parts[0], parts[1])
				}
			}

			// config 로드
			cfg, err := config.Load(args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(exitScenarioError)
			}

			// CLI 플래그로 오버라이드
			if vusersOverride > 0 && len(cfg.Scenarios) > 0 {
				cfg.Scenarios[0].VUsers = vusersOverride
			}
			if durationOverride != "" && len(cfg.Scenarios) > 0 {
				d, err := time.ParseDuration(durationOverride)
				if err != nil {
					fmt.Fprintf(os.Stderr, "✗ Error: invalid duration %q: %v\n", durationOverride, err)
					os.Exit(exitScenarioError)
				}
				cfg.Scenarios[0].Duration = d
			}
			if rampUpOverride != "" && len(cfg.Scenarios) > 0 {
				d, err := time.ParseDuration(rampUpOverride)
				if err != nil {
					fmt.Fprintf(os.Stderr, "✗ Error: invalid ramp-up %q: %v\n", rampUpOverride, err)
					os.Exit(exitScenarioError)
				}
				cfg.Scenarios[0].RampUp = d
			}

			// signal 핸들링
			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			// 테스트 실행
			result, err := runner.Run(ctx, cfg, runner.Options{
				Quiet:   quiet,
				NoColor: noColor,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Error: %v\n", err)
				os.Exit(exitConnectionErr)
			}

			// threshold 평가
			result.ThresholdResults = threshold.Evaluate(cfg.Thresholds, result)

			// 출력
			printer := output.NewPrinter(nil, noColor, 0)

			fmt.Println()
			fmt.Println("✓ Test completed.")

			printer.PrintSummary(result)
			printer.PrintThresholds(result.ThresholdResults)

			// 리포트 생성
			if len(outputFormats) > 0 {
				if err := report.Generate(result, outputFormats, outDir); err != nil {
					fmt.Fprintf(os.Stderr, "✗ Error generating report: %v\n", err)
					os.Exit(exitInternalError)
				}
			}

			// exit code
			for _, t := range result.ThresholdResults {
				if !t.Passed {
					os.Exit(exitThresholdFail)
				}
			}

			return nil
		},
	}

	runCmd.Flags().IntVar(&vusersOverride, "vusers", 0, "Override virtual users count")
	runCmd.Flags().StringVar(&durationOverride, "duration", "", "Override test duration (e.g., \"5m\", \"30s\")")
	runCmd.Flags().StringVar(&rampUpOverride, "ramp-up", "", "Override ramp-up period")
	runCmd.Flags().StringSliceVar(&outputFormats, "out", nil, "Output format: json, html (repeatable)")
	runCmd.Flags().StringVar(&outDir, "out-dir", "./reports", "Output directory for reports")
	runCmd.Flags().BoolVar(&quiet, "quiet", false, "Show only final summary")
	runCmd.Flags().StringSliceVar(&envVars, "env", nil, "Set environment variable KEY=VALUE (repeatable)")

	// validate command
	validateCmd := &cobra.Command{
		Use:   "validate <scenario.yaml>",
		Short: "Validate a YAML scenario file without running",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.Load(args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(exitScenarioError)
			}
			fmt.Println("✓ Scenario file is valid.")
			return nil
		},
	}

	// version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("omnitest %s\n", version)
		},
	}

	// --- Cycle 2: controller command ---
	var (
		ctrlGRPCPort int
		ctrlHTTPPort int
		ctrlDBURL    string
	)

	controllerCmd := &cobra.Command{
		Use:   "controller",
		Short: "Start the OmniTest Controller server",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 환경변수 fallback
			if ctrlDBURL == "" {
				ctrlDBURL = os.Getenv("OMNITEST_DB_URL")
			}
			if ctrlDBURL == "" {
				ctrlDBURL = "postgres://omnitest:omnitest@localhost:5432/omnitest?sslmode=disable"
			}
			if p := os.Getenv("OMNITEST_GRPC_PORT"); p != "" && ctrlGRPCPort == 9090 {
				fmt.Sscanf(p, "%d", &ctrlGRPCPort)
			}
			if p := os.Getenv("OMNITEST_HTTP_PORT"); p != "" && ctrlHTTPPort == 8080 {
				fmt.Sscanf(p, "%d", &ctrlHTTPPort)
			}

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			ctrl, err := controller.New(ctx, controller.Config{
				GRPCPort:    ctrlGRPCPort,
				HTTPPort:    ctrlHTTPPort,
				DatabaseURL: ctrlDBURL,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Failed to create controller: %v\n", err)
				os.Exit(exitInternalError)
			}

			apiServer := api.NewServer(ctrl)

			fmt.Printf("→ Controller started\n")
			fmt.Printf("  gRPC: :%d\n", ctrlGRPCPort)
			fmt.Printf("  HTTP: :%d\n", ctrlHTTPPort)

			go func() {
				<-ctx.Done()
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer shutdownCancel()
				ctrl.Shutdown(shutdownCtx)
			}()

			if err := ctrl.Start(ctx, apiServer.Handler()); err != nil {
				fmt.Fprintf(os.Stderr, "✗ Controller error: %v\n", err)
				os.Exit(exitInternalError)
			}

			return nil
		},
	}

	controllerCmd.Flags().IntVar(&ctrlGRPCPort, "grpc-port", 9090, "gRPC server port")
	controllerCmd.Flags().IntVar(&ctrlHTTPPort, "http-port", 8080, "HTTP/REST API server port")
	controllerCmd.Flags().StringVar(&ctrlDBURL, "db-url", "", "PostgreSQL database URL")

	// --- Cycle 2: agent command ---
	var (
		agentController     string
		agentControllerHTTP string
		agentName           string
		agentMaxVUsers      int
	)

	agentCmd := &cobra.Command{
		Use:   "agent",
		Short: "Start the OmniTest Agent mode (connects to controller)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if agentController == "" {
				agentController = os.Getenv("OMNITEST_CONTROLLER_ADDR")
			}
			if agentController == "" {
				fmt.Fprintln(os.Stderr, "✗ Error: --controller flag is required")
				os.Exit(exitScenarioError)
			}
			if agentControllerHTTP == "" {
				agentControllerHTTP = os.Getenv("OMNITEST_CONTROLLER_HTTP")
			}
			if agentName == "" {
				agentName = os.Getenv("OMNITEST_AGENT_NAME")
			}
			if p := os.Getenv("OMNITEST_MAX_VUSERS"); p != "" && agentMaxVUsers == 1000 {
				fmt.Sscanf(p, "%d", &agentMaxVUsers)
			}

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			a := agent.New(agent.Config{
				ControllerAddr: agentController,
				ControllerHTTP: agentControllerHTTP,
				Name:           agentName,
				MaxVUsers:      agentMaxVUsers,
			})

			if err := a.Run(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "✗ Agent error: %v\n", err)
				os.Exit(exitConnectionErr)
			}

			return nil
		},
	}

	agentCmd.Flags().StringVar(&agentController, "controller", "", "Controller gRPC address (host:port)")
	agentCmd.Flags().StringVar(&agentControllerHTTP, "controller-http", "", "Controller HTTP address (e.g. http://controller:8080)")
	agentCmd.Flags().StringVar(&agentName, "name", "", "Agent name (default: hostname)")
	agentCmd.Flags().IntVar(&agentMaxVUsers, "max-vusers", 1000, "Maximum VUsers this agent can handle")

	rootCmd.AddCommand(runCmd, validateCmd, versionCmd, controllerCmd, agentCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(exitInternalError)
	}
}
