// Package main provides CLI to validate AWS infrastructure.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/brunojet/go-edge-cache/internal/infra"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	validateCmd := flag.NewFlagSet("validate", flag.ExitOnError)
	profile := validateCmd.String("profile", "", "AWS profile to use")
	region := validateCmd.String("region", "", "AWS region to use")

	if len(os.Args) < 2 {
		fmt.Println("Usage: infra <command>\nCommands:\n  validate   Validate AWS credentials and list buckets")
		return fmt.Errorf("no command specified")
	}

	switch os.Args[1] {
	case "validate":
		_ = validateCmd.Parse(os.Args[2:])
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := infra.Init(); err != nil {
			return fmt.Errorf("infra init failed: %w", err)
		}

		res, err := infra.ValidateAWS(ctx, *profile, *region)
		if err != nil {
			return fmt.Errorf("AWS validation failed: %w", err)
		}

		fmt.Printf("Account: %s\nARN: %s\nUserID: %s\nRegion: %s\nBuckets: %v\n", res.Account, res.ARN, res.UserID, res.Region, res.Buckets)
		return nil

	default:
		return fmt.Errorf("unknown command: %s", os.Args[1])
	}
}
