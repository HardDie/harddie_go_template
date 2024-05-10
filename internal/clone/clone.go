package clone

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"

	"github.com/HardDie/harddie_go_template/internal/clone/edit"
	"github.com/HardDie/harddie_go_template/internal/logger"
)

// ErrInvalidParams signals about app misusage.
var ErrInvalidParams = errors.New("invalid params")

// Applicator builds new applications.
type Applicator struct {
}

// New creates Applicator instance.
func New() *Applicator {
	return &Applicator{}
}

// Run command, accept CLI arguments.
func (a *Applicator) Run(args []string) error {
	if len(args) == 0 {
		return ErrInvalidParams
	}
	command := args[0]
	switch command {
	case "create":
		return a.Create(args[1:])
	}
	return ErrInvalidParams
}

// Create new application. It downloads dev-onboarding@latest and replace dev-onboarding with provided appName.
func (a *Applicator) Create(args []string) error {
	if len(args) != 2 {
		return ErrInvalidParams
	}
	srcMod := "github.com/HardDie/harddie_go_template"
	srcModVers := srcMod + "@latest"
	if err := module.CheckPath(srcMod); err != nil {
		return fmt.Errorf("invalid source module name: %v", err)
	}

	appName := args[0]
	if appName == "" {
		return ErrInvalidParams
	}

	dstMod := fmt.Sprintf("github.com/HardDie/%s", appName)
	if err := module.CheckPath(dstMod); err != nil {
		return fmt.Errorf("invalid destination module name: %v", err)
	}

	dir, err := filepath.Abs(args[1])
	if err != nil {
		return fmt.Errorf("bad path at dir arg: %w", err)
	}

	// Dir must not exist or must be an empty directory.
	de, err := os.ReadDir(dir)
	if err == nil && len(de) > 0 {
		return fmt.Errorf("target directory %s exists and is non-empty", dir)
	}
	needMkdir := err != nil

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "mod", "download", "-json", srcModVers)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod download -json %s: %v\n%s%s", srcModVers, err, stderr.Bytes(), stdout.Bytes())
	}

	var info struct {
		Dir string
	}
	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		return fmt.Errorf("go mod download -json %s: invalid JSON output: %v\n%s%s", srcMod, err, stderr.Bytes(), stdout.Bytes())
	}

	if needMkdir {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return err
		}
	}

	// Copy from module cache into new directory, making edits as needed.
	err = filepath.WalkDir(info.Dir, func(src string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(info.Dir, src)
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, rel)
		if d.IsDir() {
			if err := os.MkdirAll(dst, 0777); err != nil {
				return err
			}
			return nil
		}

		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}

		isRoot := !strings.Contains(rel, string(filepath.Separator))
		if strings.HasSuffix(rel, ".go") {
			data, err = fixGo(data, rel, srcMod, dstMod, isRoot)
			if err != nil {
				return err
			}
		} else {
			data = fixConfiguration(data, appName)
		}
		if rel == "go.mod" {
			data, err = fixGoMod(data, dstMod)
			if err != nil {
				return err
			}
		}
		if err := os.WriteFile(dst, data, 0666); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("initialized %s in %s", dstMod, dir))

	return nil
}

// fixGo rewrites the Go source in data to replace srcMod with dstMod.
// isRoot indicates whether the file is in the root directory of the module,
// in which case we also update the package name.
func fixGo(data []byte, file string, srcMod, dstMod string, isRoot bool) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, data, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parsing source module: %s", err)
	}

	buf := edit.NewBuffer(data)
	at := func(p token.Pos) int {
		return fset.File(p).Offset(p)
	}

	srcName := path.Base(srcMod)
	dstName := path.Base(dstMod)
	if isRoot {
		if name := f.Name.Name; name == srcName || name == srcName+"_test" {
			dname := dstName + strings.TrimPrefix(name, srcName)
			if !token.IsIdentifier(dname) {
				return nil, fmt.Errorf("%s: cannot rename package %s to package %s: invalid package name", file, name, dname)
			}
			buf.Replace(at(f.Name.Pos()), at(f.Name.End()), dname)
		}
	}

	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		if path == srcMod {
			if srcName != dstName && spec.Name == nil {
				// Add package rename because source code uses original name.
				// The renaming looks strange, but template authors are unlikely to
				// create a template where the root package is imported by packages
				// in subdirectories, and the renaming at least keeps the code working.
				// A more sophisticated approach would be to rename the uses of
				// the package identifier in the file too, but then you have to worry about
				// name collisions, and given how unlikely this is, it doesn't seem worth
				// trying to clean up the file that way.
				buf.Insert(at(spec.Path.Pos()), srcName+" ")
			}
			// Change import path to dstMod
			buf.Replace(at(spec.Path.Pos()), at(spec.Path.End()), strconv.Quote(dstMod))
		}
		if strings.HasPrefix(path, srcMod+"/") {
			// Change import path to begin with dstMod
			buf.Replace(at(spec.Path.Pos()), at(spec.Path.End()), strconv.Quote(strings.Replace(path, srcMod, dstMod, 1)))
		}
	}
	return buf.Bytes(), nil
}

// fixGoMod rewrites the go.mod content in data to replace srcMod with dstMod
// in the module path.
func fixGoMod(data []byte, dstMod string) ([]byte, error) {
	f, err := modfile.ParseLax("go.mod", data, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing source module:%s", err)
	}
	_ = f.AddModuleStmt(dstMod)
	new, err := f.Format()
	if err != nil {
		return data, nil
	}
	return new, nil
}

// fixConfiguration replaces `dev-onboarding` with appName in all no-go files
func fixConfiguration(data []byte, appName string) []byte {
	result := strings.ReplaceAll(string(data), "harddie_go_template", appName)
	return []byte(result)
}
