package cli

import (
	"flag"
	"fmt"
	"strings"

	"iterforge/internal/templates"
)

// InitProject scaffolds a new IterForge project from a workflow template.
func InitProject(args []string) int {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	parent := fs.String("dir", ".", "parent directory to create the project in")
	tmpl := fs.String("template", templates.DefaultTemplate,
		"workflow template ("+strings.Join(templates.Available(), ", ")+")")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: iterforge init [-dir <parent>] [-template <name>] <project-name>")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if fs.NArg() != 1 {
		fs.Usage()
		return 1
	}
	name := fs.Arg(0)

	if err := templates.Init(name, *parent, *tmpl); err != nil {
		return errExit(err)
	}

	fmt.Printf("created IterForge project %q (template %q) in %s\n", name, *tmpl, *parent)
	fmt.Printf("next:\n  cd %s\n  make check && make baseline && make summarize\n", name)
	return 0
}
