// Copyright 2020 Hajime Hoshi
// SPDX-License-Identifier: Apache-2.0

// +build ignore

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	if err := genStdfiles(); err != nil {
		return err
	}
	if err := genBins(); err != nil {
		return err
	}
	return nil
}

const (
	goversion = "1.14beta1"
	goname    = "go" + goversion
)

func stdfiles() (string, []string, error) {
	var src string
	{
		cmd := exec.Command(goname, "list", "-f", "{{.Dir}}", "runtime")
		cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
		cmd.Stderr = os.Stderr
		out, err := cmd.Output()
		if err != nil {
			return "", nil, err
		}
		src = filepath.Join(strings.TrimSpace(string(out)), "..")
	}

	cmd := exec.Command(goname, "list", "-f", "dir: {{.Dir}}\n{{range .GoFiles}}file: {{.}}\n{{end}}{{range .SFiles}}file: {{.}}\n{{end}}", "std")
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", nil, err
	}

	var files []string
	var dir string
	for _, line := range strings.Split(string(out), "\n") {
		const predir = "dir:"
		const prefile = "file:"
		if strings.HasPrefix(line, predir) {
			dir = strings.TrimSpace(line[len(predir):])
			continue
		}
		if strings.HasPrefix(line, prefile) {
			file := strings.TrimSpace(line[len(prefile):])
			rel, err := filepath.Rel(src, filepath.Join(dir, file))
			if err != nil {
				return "", nil, err
			}
			files = append(files, rel)
		}
	}
	return src, files, nil
}

func genStdfiles() error {
	src, fs, err := stdfiles()
	if err != nil {
		return err
	}

	contents := map[string]string{}
	for _, f := range fs {
		c, err := ioutil.ReadFile(filepath.Join(src, f))
		if err != nil {
			return err
		}
		contents[f] = base64.StdEncoding.EncodeToString(c)
	}

	f, err := os.OpenFile("stdfiles.js", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "// Copyright 2020 Hajime Hoshi")
	fmt.Fprintln(f, "// SPDX-License-Identifier: Apache-2.0")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "// Code generated by gen.go. DO NOT EDIT.")
	fmt.Fprintln(f)

	fmt.Fprint(f, "export const stdfiles = ")
	e := json.NewEncoder(f)
	if err := e.Encode(contents); err != nil {
		return err
	}
	return nil
}

func genBins() error {
	files := []struct {
		Name string
		Path string
	}{
		{
			Name: "go" + goversion + ".wasm",
			Path: "cmd/go",
		},
		{
			Name: "compile" + goversion + ".wasm",
			Path: "cmd/compile",
		},
		{
			Name: "link" + goversion + ".wasm",
			Path: "cmd/link",
		},
	}
	for _, file := range files {
		cmd := exec.Command(goname, "build", "-trimpath", "-o=bin/"+file.Name, file.Path)
		cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}
