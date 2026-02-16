package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "TEST_POSTGRES_DSN is required")
		os.Exit(2)
	}

	timeout := 60 * time.Second
	if raw := os.Getenv("WAIT_FOR_POSTGRES_TIMEOUT_SEC"); raw != "" {
		secs, err := strconv.Atoi(raw)
		if err != nil || secs <= 0 {
			fmt.Fprintf(os.Stderr, "invalid WAIT_FOR_POSTGRES_TIMEOUT_SEC: %q\n", raw)
			os.Exit(2)
		}
		timeout = time.Duration(secs) * time.Second
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open postgres: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	deadline := time.Now().Add(timeout)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := db.PingContext(ctx)
		cancel()
		if err == nil {
			fmt.Println("postgres ready")
			return
		}
		if time.Now().After(deadline) {
			fmt.Fprintf(os.Stderr, "postgres not ready within %s: %v\n", timeout, err)
			os.Exit(1)
		}
		time.Sleep(2 * time.Second)
	}
}
