package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/crazyinfin8/wagu/gen"
	"github.com/crazyinfin8/wagu/ir"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"golang.org/x/tools/imports"
)

var (
	genOutDir           string
	genPkg              string
	genExprComments     bool
	genCamelCaseExports bool
	genExportExports    bool
	genMMap             bool
	genUseUnsafe        bool
	genCmd              = &cobra.Command{
		Use:   "gen FILE.ir",
		Short: "Convert IR to source code",
		Args:  cobra.ExactArgs(1),
		RunE:  genSrc,
	}
)

func init() {
	genCmd.Flags().StringVarP(&genOutDir, "output_dir", "d", "gen", "output dir")
	genCmd.Flags().StringVarP(&genPkg, "pkg", "p", "gen", "package name")
	genCmd.Flags().BoolVarP(&genExprComments, "expr_comments", "C", false, "whether to emit expr comments")
	genCmd.Flags().BoolVarP(&genCamelCaseExports, "camel_case_exports", "c", false, "whether to camel case exports")
	genCmd.Flags().BoolVarP(&genExportExports, "export_exports", "e", true, "whether to export exports")
	genCmd.Flags().BoolVarP(&genMMap, "mmap", "m", false, "mmap memory")
	genCmd.Flags().BoolVarP(&genUseUnsafe, "use_unsafe", "u", false, "use unsafe for memory ops, indirect calls etc")
}

func genSrc(cmd *cobra.Command, args []string) error {
	b, err := ioutil.ReadFile(args[0])
	if err != nil {
		return err
	}
	m := new(ir.Module)
	if err := proto.Unmarshal(b, m); err != nil {
		return err
	}
	_, err = os.Stat(genOutDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(genOutDir, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	b, err = gen.Context(genPkg, m, genMMap, genUseUnsafe)
	if err != nil {
		return err
	}
	if err := write("context", b); err != nil {
		return err
	}
	if len(m.ImportedFunctions) != 0 {
		b, err = gen.ImportsInterface(genPkg, m)
		if err != nil {
			return err
		}
		if err := write("imports", b); err != nil {
			return err
		}
		for i, f := range m.ImportedFunctions {
			b, err = gen.ImportedFunc(genPkg, i, f)
			if err != nil {
				return err
			}
			if err := write(fmt.Sprintf("f%d", i), b); err != nil {
				return err
			}
		}
	}
	if len(m.ExportedFunctions) != 0 {
		b, err = gen.Exports(genPkg, m, genCamelCaseExports, genExportExports)
		if err != nil {
			return err
		}
		if err := write("exports", b); err != nil {
			return err
		}
	}
	for _, f := range m.Functions {
		b, err = gen.Func(genPkg, f, m.Globals, genExprComments, genMMap, genUseUnsafe)
		if err != nil {
			return err
		}
		if err := write(fmt.Sprintf("f%d", f.Id), b); err != nil {
			return err
		}
	}
	if m.Memory.Size != 0 {
		b, err = gen.Mem(genPkg, m.Memory, genMMap)
		if err != nil {
			return err
		}
		if err := write("mem", b); err != nil {
			return err
		}
	}
	return nil
}

func write(filename string, source []byte) error {
	fn := filepath.Join(genOutDir, filename+".go")
	var err error
	source, err = imports.Process(fn, source, nil)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fn, source, 0644)
}
