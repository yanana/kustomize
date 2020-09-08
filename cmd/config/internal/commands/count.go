// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0
//
package commands

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/cmd/config/ext"
	"sigs.k8s.io/kustomize/cmd/config/internal/generateddocs/commands"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func GetCountRunner(name string) *CountRunner {
	r := &CountRunner{}
	c := &cobra.Command{
		Use:     "count [DIR]",
		Args:    cobra.MaximumNArgs(1),
		Short:   commands.CountShort,
		Long:    commands.CountLong,
		Example: commands.CountExamples,
		RunE:    r.runE,
	}
	fixDocs(name, c)
	c.Flags().BoolVar(&r.Kind, "kind", true,
		"count resources by kind.")
	c.Flags().BoolVarP(&r.RecurseSubPackages, "recurse-subpackages", "R", true,
		"prints count of resources recursively in all the nested subpackages")
	r.Command = c
	return r
}

func CountCommand(name string) *cobra.Command {
	return GetCountRunner(name).Command
}

// CountRunner contains the run function
type CountRunner struct {
	IncludeSubpackages bool
	Kind               bool
	Command            *cobra.Command
	RecurseSubPackages bool
}

func (r *CountRunner) runE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		input := &kio.ByteReader{Reader: c.InOrStdin()}

		return handleError(c, kio.Pipeline{
			Inputs:  []kio.Reader{input},
			Outputs: r.out(c.OutOrStdout()),
		}.Execute())
	}

	e := executeCmdOnPkgs{
		writer:             c.OutOrStdout(),
		needOpenAPI:        false,
		recurseSubPackages: r.RecurseSubPackages,
		cmdRunner:          r,
		rootPkgPath:        args[0],
	}

	return e.execute()
}

func (r *CountRunner) executeCmd(w io.Writer, pkgPath string) error {
	openAPIFileName, err := ext.OpenAPIFileName()
	if err != nil {
		return err
	}

	input := kio.LocalPackageReader{PackagePath: pkgPath, PackageFileName: openAPIFileName}

	fmt.Fprintf(w, "%q:\n", pkgPath)
	err = kio.Pipeline{
		Inputs:  []kio.Reader{input},
		Outputs: r.out(w),
	}.Execute()

	if err != nil {
		// return err if there is only package
		if !r.RecurseSubPackages {
			return err
		} else {
			// print error message and continue if there are multiple packages to annotate
			fmt.Fprintf(w, "%s in package %q\n", err.Error(), pkgPath)
		}
	}
	return nil
}

func (r *CountRunner) out(w io.Writer) []kio.Writer {
	var out []kio.Writer
	if r.Kind {
		out = append(out, kio.WriterFunc(func(nodes []*yaml.RNode) error {
			count := map[string]int{}
			k := sets.String{}
			for _, n := range nodes {
				m, _ := n.GetMeta()
				count[m.Kind]++
				k.Insert(m.Kind)
			}
			order := k.List()
			sort.Strings(order)
			for _, k := range order {
				fmt.Fprintf(w, "%s: %d\n", k, count[k])
			}
			return nil
		}))
	} else {
		out = append(out, kio.WriterFunc(func(nodes []*yaml.RNode) error {
			fmt.Fprintf(w, "%d\n", len(nodes))
			return nil
		}))
	}
	return out
}
