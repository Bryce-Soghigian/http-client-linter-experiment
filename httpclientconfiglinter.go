package main

import (
	"crypto/tls"
	"fmt"
	"go/ast"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/singlechecker"
	"golang.org/x/tools/go/packages"
	"net"
	"net/http"
	"reflect"
	"time"
)

// Idea behind linter, it kind of works?

var httpConfig = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	},
}

func main1() {
	analyzer := &analysis.Analyzer{
		Name: "httpclientconfig",
		Doc:  "check if all instances of http.Client have the same configuration",
		Run:  run,
		Requires: []*analysis.Analyzer{
			inspect.Analyzer,
		},
	}
	singlechecker.Main(analyzer)
}

func run(pass *analysis.Pass) (interface{}, error) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax}, pass.Pkg.Path())
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				if c, ok := n.(*ast.CompositeLit); ok {
					if t, ok := c.Type.(*ast.SelectorExpr); ok {
						if x, ok := t.X.(*ast.Ident); ok && x.Name == "http" && t.Sel.Name == "Client" {
							client := reflect.New(reflect.TypeOf(http.Client{})).Elem().Interface()
							if CompareStructs(client, c) {
								// handle the case where c is the same as the reference configuration
							} else {
								// handle the case where c is different than the reference configuration
								fmt.Println("Struct does not match")
							}
						}
					}
				}
				return true
			})
		}
	}

	return nil, nil
}

// CompareStructs recursively compares the values of two structs
func CompareStructs(a, b interface{}) bool {
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	if va.Kind() != reflect.Struct || vb.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < va.NumField(); i++ {
		fa := va.Field(i)
		fb := vb.Field(i)

		switch fa.Kind() {
		case reflect.Struct:
			if !CompareStructs(fa.Interface(), fb.Interface()) {
				return false
			}
		case reflect.Ptr:
			if fa.IsNil() != fb.IsNil() {
				return false
			}
			if !fa.IsNil() && !CompareStructs(fa.Elem().Interface(), fb.Elem().Interface()) {
				return false
			}
		default:
			if !reflect.DeepEqual(fa.Interface(), fb.Interface()) {
				return false
			}
		}
	}

	return true
}
