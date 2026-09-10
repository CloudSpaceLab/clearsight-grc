//go:build postgres

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	var artifactID, reason string
	flag.StringVar(&artifactID, "artifact", "", "failed capture artifact UUID")
	flag.StringVar(&reason, "reason", "", "operator recovery reason")
	flag.Parse()
	if artifactID == "" || reason == "" {
		fatal(fmt.Errorf("-artifact and -reason are required"))
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		fatal(fmt.Errorf("DATABASE_URL is required"))
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fatal(err)
	}
	defer pool.Close()
	if err := evidence.NewPostgresRepository(pool).RequeueFailedArtifactScan(ctx, artifactID, reason, time.Now().UTC()); err != nil {
		fatal(err)
	}
	fmt.Println("artifact inspection requeued")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
