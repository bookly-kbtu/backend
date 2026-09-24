// Package command is the CLI transport: parses args and calls usecases,
// the same role transport/rest plays for HTTP.
package command

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bookly-kbtu/backend/internal/domain"
	importeruc "github.com/bookly-kbtu/backend/internal/usecase/importer"
)

const usage = `usage: go run ./cmd/command <command> [flags]

commands:
  sources                      list registered import sources
  cities  -source zapis_kz     list cities of a source (live request)
  import  -source zapis_kz -city 1[,2] | -all-cities [-max-firms N] [-snapshots]
                               import firms, services and masters
  runs    [-limit 20]          list recent import runs

run "<command> -h" for flags`

// Exit codes.
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

type CLI struct {
	importer *importeruc.Service
	out      io.Writer
	errOut   io.Writer
}

func New(importer *importeruc.Service, out, errOut io.Writer) *CLI {
	return &CLI{importer: importer, out: out, errOut: errOut}
}

// Run executes one command and returns the process exit code.
func (c *CLI) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(c.errOut, usage)
		return exitUsage
	}

	var err error
	switch args[0] {
	case "sources":
		err = c.sources()
	case "cities":
		err = c.cities(ctx, args[1:])
	case "import":
		err = c.importRun(ctx, args[1:])
	case "runs":
		err = c.runs(ctx, args[1:])
	case "-h", "--help", "help":
		fmt.Fprintln(c.out, usage)
		return exitOK
	default:
		fmt.Fprintf(c.errOut, "unknown command %q\n\n%s\n", args[0], usage)
		return exitUsage
	}

	switch {
	case err == nil:
		return exitOK
	case errors.Is(err, flag.ErrHelp):
		return exitOK
	case errors.Is(err, errUsage), errors.Is(err, domain.ErrValidation):
		fmt.Fprintln(c.errOut, "error:", err)
		return exitUsage
	default:
		fmt.Fprintln(c.errOut, "error:", err)
		return exitError
	}
}

var errUsage = errors.New("invalid usage")

func (c *CLI) flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(c.errOut)
	return fs
}

func (c *CLI) sources() error {
	w := c.table()
	fmt.Fprintln(w, "CODE\tNAME\tBASE URL")
	for _, s := range c.importer.Sources() {
		fmt.Fprintf(w, "%s\t%s\t%s\n", s.Code, s.Name, s.BaseURL)
	}
	return w.Flush()
}

func (c *CLI) cities(ctx context.Context, args []string) error {
	fs := c.flagSet("cities")
	source := fs.String("source", "", "source code (see `sources`)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *source == "" {
		return fmt.Errorf("%w: -source is required", errUsage)
	}

	cities, err := c.importer.Cities(ctx, *source)
	if err != nil {
		return err
	}

	w := c.table()
	fmt.Fprintln(w, "ID\tSLUG\tNAME")
	for _, ct := range cities {
		fmt.Fprintf(w, "%s\t%s\t%s\n", ct.ExternalID, ct.Slug, ct.Name)
	}
	return w.Flush()
}

func (c *CLI) importRun(ctx context.Context, args []string) error {
	fs := c.flagSet("import")
	source := fs.String("source", "", "source code (see `sources`)")
	cities := fs.String("city", "", "comma-separated city IDs of the source (see `cities`)")
	allCities := fs.Bool("all-cities", false, "import every city of the source")
	maxFirms := fs.Int("max-firms", 0, "max firms per city, 0 = all")
	snapshots := fs.Bool("snapshots", false, "store raw responses in source_snapshots")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *source == "" {
		return fmt.Errorf("%w: -source is required", errUsage)
	}

	run, err := c.importer.Run(ctx, importeruc.RunInput{
		SourceCode:    *source,
		CityIDs:       splitList(*cities),
		AllCities:     *allCities,
		MaxFirms:      *maxFirms,
		SaveSnapshots: *snapshots,
	})
	if run != nil {
		c.printRun(run)
	}
	if err != nil {
		return err
	}
	if run.Status != domain.ImportRunCompleted {
		return fmt.Errorf("run %s", run.Status)
	}
	return nil
}

func (c *CLI) runs(ctx context.Context, args []string) error {
	fs := c.flagSet("runs")
	limit := fs.Int("limit", 20, "number of runs")
	if err := fs.Parse(args); err != nil {
		return err
	}

	runs, err := c.importer.Runs(ctx, *limit)
	if err != nil {
		return err
	}

	w := c.table()
	fmt.Fprintln(w, "STARTED\tSOURCE\tSTATUS\tDURATION\tCITIES\tFIRMS\tMASTERS\tSERVICES\tERRORS\tID")
	for _, r := range runs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s\n",
			r.StartedAt.Local().Format("2006-01-02 15:04"), r.SourceCode, r.Status, duration(r),
			r.CitiesProcessed, r.FirmsProcessed, r.MastersProcessed, r.ServicesProcessed, r.ErrorsCount, r.ID)
	}
	return w.Flush()
}

func (c *CLI) printRun(r *domain.ImportRun) {
	w := c.table()
	fmt.Fprintf(w, "run\t%s\n", r.ID)
	fmt.Fprintf(w, "status\t%s\n", r.Status)
	fmt.Fprintf(w, "duration\t%s\n", duration(*r))
	fmt.Fprintf(w, "cities\t%d\n", r.CitiesProcessed)
	fmt.Fprintf(w, "firms\t%d\n", r.FirmsProcessed)
	fmt.Fprintf(w, "masters\t%d\n", r.MastersProcessed)
	fmt.Fprintf(w, "services\t%d\n", r.ServicesProcessed)
	fmt.Fprintf(w, "errors\t%d\n", r.ErrorsCount)
	_ = w.Flush()

	if r.ErrorSummary != nil {
		fmt.Fprintln(c.out, "\nerrors (first 20):")
		fmt.Fprintln(c.out, *r.ErrorSummary)
	}
}

func (c *CLI) table() *tabwriter.Writer {
	return tabwriter.NewWriter(c.out, 0, 0, 2, ' ', 0)
}

func duration(r domain.ImportRun) string {
	if r.FinishedAt == nil {
		return "-"
	}
	return r.FinishedAt.Sub(r.StartedAt).Round(time.Second).String()
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
