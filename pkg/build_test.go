// Copyright The SLSA team.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pkg

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func errCmp(e1, e2 error) bool {
	return errors.Is(e1, e2) || errors.Is(e2, e1)
}

func Test_marshallList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		variables []string
		expected  string
	}{
		{
			name:      "single arg",
			variables: []string{"--arg"},
			expected:  "WyItLWFyZyJd",
		},
		{
			name: "list args",
			variables: []string{
				"/usr/lib/google-golang/bin/go",
				"build", "-mod=vendor", "-trimpath",
				"-tags=netgo",
				"-ldflags=-X main.gitVersion=v1.2.3 -X main.gitSomething=somthg",
			},
			expected: "WyIvdXNyL2xpYi9nb29nbGUtZ29sYW5nL2Jpbi9nbyIsImJ1aWxkIiwiLW1vZD12ZW5kb3IiLCItdHJpbXBhdGgiLCItdGFncz1uZXRnbyIsIi1sZGZsYWdzPS1YIG1haW4uZ2l0VmVyc2lvbj12MS4yLjMgLVggbWFpbi5naXRTb21ldGhpbmc9c29tdGhnIl0=",
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := marshallList(tt.variables)
			if err != nil {
				t.Errorf("marshallList: %v", err)
			}
			if !cmp.Equal(r, tt.expected) {
				t.Errorf(cmp.Diff(r, tt.expected))
			}
		})
	}
}

func Test_generateCommandEnvVariables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      []string
		expected struct {
			err   error
			flags []string
		}
	}{
		{
			name: "valid env GO",
			env:  []string{"GOOS=linux", "GOARCH=x86"},
			expected: struct {
				err   error
				flags []string
			}{
				flags: []string{"GOOS=linux", "GOARCH=x86"},
				err:   nil,
			},
		},
		{
			name: "valid env",
			env:  []string{"VAR1=value1", "VAR2=value2"},
			expected: struct {
				err   error
				flags []string
			}{
				flags: []string{"VAR1=value1", "VAR2=value2"},
			},
		},
		{
			name: "more valid flags",
			env:  []string{"GOVAR1=value1", "GOVAR2=value2", "CGO_VAR1=val1", "CGO_VAR2=val2"},
			expected: struct {
				err   error
				flags []string
			}{
				flags: []string{
					"GOVAR1=value1", "GOVAR2=value2", "CGO_VAR1=val1", "CGO_VAR2=val2",
				},
				err: nil,
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := KoBuildNew("go compiler")

			err := b.SetArgEnvVariables(strings.Join(tt.env, ","))
			if err != nil {
				t.Fatal(fmt.Sprintf("SetArgEnvVariables error: %v", err))
			}

			flags, err := b.generateEnvVariables()

			if !errCmp(err, tt.expected.err) {
				t.Errorf(cmp.Diff(err, tt.expected.err))
			}
			if err != nil {
				return
			}
			// Note: generated env variables contain the process's env variables too.
			expectedFlags := append(os.Environ(), tt.expected.flags...)
			sorted := cmpopts.SortSlices(func(a, b string) bool { return a < b })
			if !cmp.Equal(flags, expectedFlags, sorted) {
				t.Errorf(cmp.Diff(flags, expectedFlags))
			}
		})
	}
}

func Test_SetArgEnvVariables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		argEnv   string
		expected struct {
			err error
			env map[string]string
		}
	}{
		{
			name:   "valid arg envs",
			argEnv: "VAR1=value1, VAR2=value2",
			expected: struct {
				err error
				env map[string]string
			}{
				err: nil,
				env: map[string]string{"VAR1": "value1", "VAR2": "value2"},
			},
		},
		{
			name:   "empty arg envs",
			argEnv: "",
			expected: struct {
				err error
				env map[string]string
			}{
				err: nil,
				env: map[string]string{},
			},
		},
		{
			name:   "valid arg envs not space",
			argEnv: "VAR1=value1,VAR2=value2",
			expected: struct {
				err error
				env map[string]string
			}{
				err: nil,
				env: map[string]string{"VAR1": "value1", "VAR2": "value2"},
			},
		},
		{
			name:   "invalid arg empty 2 values",
			argEnv: "VAR1=value1,",
			expected: struct {
				err error
				env map[string]string
			}{
				err: errorInvalidEnvArgument,
			},
		},
		{
			name:   "invalid arg empty 3 values",
			argEnv: "VAR1=value1,, VAR3=value3",
			expected: struct {
				err error
				env map[string]string
			}{
				err: errorInvalidEnvArgument,
			},
		},
		{
			name:   "invalid arg uses :",
			argEnv: "VAR1:value1",
			expected: struct {
				err error
				env map[string]string
			}{
				err: errorInvalidEnvArgument,
			},
		},
		{
			name:   "valid single arg",
			argEnv: "VAR1=value1",
			expected: struct {
				err error
				env map[string]string
			}{
				err: nil,
				env: map[string]string{"VAR1": "value1"},
			},
		},
		{
			name:   "invalid valid single arg with empty",
			argEnv: "VAR1=value1=",
			expected: struct {
				err error
				env map[string]string
			}{
				err: errorInvalidEnvArgument,
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := KoBuildNew("go compiler")

			err := b.SetArgEnvVariables(tt.argEnv)
			if !errCmp(err, tt.expected.err) {
				t.Errorf(cmp.Diff(err, tt.expected.err))
			}

			if err != nil {
				return
			}

			sorted := cmpopts.SortSlices(func(a, b string) bool { return a < b })
			if !cmp.Equal(b.envs, tt.expected.env, sorted) {
				t.Errorf(cmp.Diff(b.envs, tt.expected.env))
			}
		})
	}
}

func Test_generateCommandArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     string
		expected []string
	}{
		{
			name:     "valid flags",
			args:     "-race -x",
			expected: []string{"-race", "-x"},
		},
		{
			name:     "valid all flags",
			args:     "-a -race -msan -asan -ldflags -linkshared",
			expected: []string{"-a", "-race", "-msan", "-asan", "-ldflags", "-linkshared"},
		},
	}

	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := KoBuildNew("ko")

			err := b.SetArgs(tt.args)
			if err != nil {
				t.Fatal(fmt.Sprintf("SetArgs failed: %v", err))
			}

			cmd, err := b.generateCommandArgs()
			if err != nil {
				t.Fatal(fmt.Sprintf("generateCommandArgs failed: %v", err))
			}
			expectedCmd := append([]string{"ko", "publish"}, tt.expected...)

			// Note: generated env variables contain the process's env variables too.
			sorted := cmpopts.SortSlices(func(a, b string) bool { return a < b })
			if !cmp.Equal(cmd, expectedCmd, sorted) {
				t.Errorf(cmp.Diff(cmd, expectedCmd))
			}
		})
	}
}

func Test_generateRegistry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
		err      error
	}{
		{
			name:     "docker registry",
			input:    "docker.io/username",
			expected: "docker.io",
		},
		{
			name:     "default docker registry",
			input:    "username",
			expected: "docker.io",
		},
		{
			name:     "ghcr registry",
			input:    "ghcr.io/username",
			expected: "ghcr.io",
		},
		{
			name:     "any registry",
			input:    "any/username",
			expected: "any",
		},
		{
			name:     "any registry with space",
			input:    " any/username ",
			expected: "any",
		},
		{
			name:  "invalid registry",
			input: "too/many/names",
			err:   errorInvalidRegistry,
		},
	}

	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := KoBuildNew("ko")

			err := b.SetArgEnvVariables("KO_DOCKER_REPO=" + tt.input)
			if err != nil {
				t.Fatal(fmt.Sprintf("SetArgEnvVariables failed: %v", err))
			}

			registry, err := b.generateRegistry()
			if !errCmp(err, tt.err) {
				t.Errorf(cmp.Diff(err, tt.err))
			}
			if err != nil {
				return
			}
			expectedRegistry := tt.expected
			if expectedRegistry != registry {
				t.Errorf(cmp.Diff(expectedRegistry, registry))
			}
		})
	}
}
