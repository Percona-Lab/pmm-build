// pmm-build
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/Percona-Lab/pmm-build/pmm-build/lib"
)

func main() {
	log.SetFlags(0)
	flag.Parse()

	for _, f := range flag.Args() {
		cfg, err := lib.LoadConfig(f)
		if err != nil {
			log.Fatal(err)
		}

		prefix := filepath.Join("_build", cfg.Name)

		var wg sync.WaitGroup
		for _, r := range cfg.Repositories {
			wg.Add(1)
			go func(r lib.Repository) {
				defer wg.Done()

				start := time.Now()
				out, err := r.Get(prefix)
				if err != nil {
					log.Print(out)
					log.Fatal(err)
				}
				log.Printf("%s cloned in %s.", r.URL, time.Since(start))
			}(r)
		}
		wg.Wait()
		log.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		fmt.Fprintf(w, "PATH\tVERSION\tBRANCH\tCOMMIT\n")
		for _, r := range cfg.Repositories {
			version, branch, commit, err := r.Describe(prefix)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Path, version, branch, commit)
		}
		if err := w.Flush(); err != nil {
			log.Fatal(err)
		}
	}
}
