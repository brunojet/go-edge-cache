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
	validateCmd := flag.NewFlagSet("validate", flag.ExitOnError)
	profile := validateCmd.String("profile", "", "AWS profile to use")
	region := validateCmd.String("region", "", "AWS region to use")

	if len(os.Args) < 2 {
		fmt.Println("Usage: infra <command>\nCommands:\n  validate   Validate AWS credentials and list buckets")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		validateCmd.Parse(os.Args[2:])
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := infra.Init(); err != nil {
			fmt.Fprintf(os.Stderr, "infra init failed: %v\n", err)
			os.Exit(1)
		}

		res, err := infra.ValidateAWS(ctx, *profile, *region)
		if err != nil {
			fmt.Fprintf(os.Stderr, "AWS validation failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Account: %s\nARN: %s\nUserID: %s\nRegion: %s\nBuckets: %v\n", res.Account, res.ARN, res.UserID, res.Region, res.Buckets)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(2)
	}
}
