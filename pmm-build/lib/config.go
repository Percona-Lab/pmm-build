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

package lib

import (
	"bytes"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func LoadConfig(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var cfg Config
	if err = yaml.Unmarshal(b, &cfg); err != nil {
		return nil, errors.WithStack(err)
	}

	for i, r := range cfg.Repositories {
		if err = r.setDefaults(); err != nil {
			return nil, err
		}
		cfg.Repositories[i] = r
	}
	return &cfg, nil
}

type Config struct {
	Name         string
	Repositories []Repository
}

type Repository struct {
	URL    string
	Branch string
	Tag    string
	Path   string
}

func (r *Repository) ref() (string, error) {
	switch {
	case r.Branch != "":
		if r.Tag != "" {
			return "", errors.Errorf("both branch (%q) and tag (%q) are specified", r.Branch, r.Tag)
		}
		return r.Branch, nil

	case r.Tag != "":
		return r.Tag, nil

	default:
		return "", errors.Errorf("both branch and tag are not specified")
	}
}

func (r *Repository) setDefaults() error {
	if r.Path == "" {
		u, err := url.Parse(r.URL)
		if err != nil {
			return errors.WithStack(err)
		}
		if u.Scheme != "https" {
			return errors.Errorf("can't make path from URL %q, use https:// prefix", r.URL)
		}
		r.Path = u.Hostname() + u.Path
	}

	if _, err := r.ref(); err != nil {
		return err
	}

	return nil
}

func (r *Repository) dir(prefix string) string {
	return filepath.Join(prefix, filepath.Base(r.Path), "src", r.Path)
}

func (r *Repository) shell(prefix string, args []string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = r.dir(prefix)
	b, err := cmd.CombinedOutput()
	b = bytes.TrimSpace(b)
	return string(b), errors.WithStack(err)
}

func (r *Repository) Get(prefix string) (string, error) {
	var out string
	cmd := exec.Command("git", "clone", r.URL, r.dir(prefix))
	b, err := cmd.CombinedOutput()
	out = string(b)
	if err != nil {
		return out, errors.WithStack(err)
	}

	ref, err := r.ref()
	if err != nil {
		return out, err
	}
	s, err := r.shell(prefix, []string{"git", "checkout", ref})
	out += "\n" + string(s)
	return out, err
}

func (r *Repository) Describe(prefix string) (version, branch, commit string, err error) {
	version, err = r.shell(prefix, []string{"git", "describe", "--tags", "--dirty", "--always"})
	if err != nil {
		return
	}

	branch, err = r.shell(prefix, []string{"git", "rev-parse", "--abbrev-ref", "HEAD"})
	if err != nil {
		return
	}
	if branch == "HEAD" {
		branch = ""
	}

	commit, err = r.shell(prefix, []string{"git", "rev-parse", "--short", "HEAD"})
	if err != nil {
		return
	}

	return
}
