// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"path"
	"reflect"

	"github.com/golang/mock/mockgen/model"

	pkg_ "github.com/redhat-marketplace/redhat-marketplace-operator/v2/pkg/utils/reconcileutils"
)

var output = flag.String("output", "", "The output file name, or empty to use stdout.")

func main() {
	flag.Parse()

	its := []struct {
		sym string
		typ reflect.Type
	}{

		{"ClientCommandRunner", reflect.TypeOf((*pkg_.ClientCommandRunner)(nil)).Elem()},
	}
	pkg := &model.Package{
		// NOTE: This behaves contrary to documented behaviour if the
		// package name is not the final component of the import path.
		// The reflect package doesn't expose the package name, though.
		Name: path.Base("github.com/redhat-marketplace/redhat-marketplace-operator/v2/pkg/utils/reconcileutils"),
	}

	for _, it := range its {
		intf, err := model.InterfaceFromInterfaceType(it.typ)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Reflection: %v\n", err)
			os.Exit(1)
		}
		intf.Name = it.sym
		pkg.Interfaces = append(pkg.Interfaces, intf)
	}

	outfile := os.Stdout
	if len(*output) != 0 {
		var err error
		outfile, err = os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open output file %q", *output)
		}
		defer func() {
			if err := outfile.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to close output file %q", *output)
				os.Exit(1)
			}
		}()
	}

	if err := gob.NewEncoder(outfile).Encode(pkg); err != nil {
		fmt.Fprintf(os.Stderr, "gob encode: %v\n", err)
		os.Exit(1)
	}
}