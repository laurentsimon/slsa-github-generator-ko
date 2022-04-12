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
	"strings"
)

var (
	errorEnvVariableNameEmpty      = errors.New("env variable empty or not set")
	errorUnsupportedArguments      = errors.New("argument not supported")
	errorInvalidEnvArgument        = errors.New("invalid env passed via argument")
	errorEnvVariableNameNotAllowed = errors.New("env variable not allowed")
	errorInvalidFilename           = errors.New("invalid filename")
	errorEmptyFilename             = errors.New("filename is not set")
)

// See `go build help`.
// `-asmflags`, `-n`, `-mod`, `-installsuffix`, `-modfile`,
// `-workfile`, `-overlay`, `-pkgdir`, `-toolexec`, `-o`,
// `-modcacherw`, `-work` not supported for now.

var allowedBuildArgs = map[string]bool{
	"-a": true, "-race": true, "-msan": true, "-asan": true,
	"-v": true, "-x": true, "-buildinfo": true,
	"-buildmode": true, "-buildvcs": true, "-compiler": true,
	"-gccgoflags": true, "-gcflags": true,
	"-ldflags": true, "-linkshared": true,
	"-tags": true, "-trimpath": true,
}

var allowedEnvVariablePrefix = map[string]bool{
	"GO": true, "CGO_": true, "KO_": true,
	"KIND_": true,
}

type KoBuild struct {
	ko     string
	argEnv map[string]string
}

func KoBuildNew(ko string) *KoBuild {
	c := KoBuild{
		ko:     ko,
		argEnv: make(map[string]string),
	}

	return &c
}

func (b *KoBuild) Run(dry bool) error {
	return nil
}

func (b *KoBuild) SetArgEnvVariables(envs string) error {
	if envs == "" {
		return nil
	}

	for _, e := range strings.Split(envs, ",") {
		s := strings.Trim(e, " ")

		sp := strings.Split(s, ":")
		if len(sp) != 2 {
			return fmt.Errorf("%w: %s", errorInvalidEnvArgument, s)
		}
		name := strings.Trim(sp[0], " ")
		value := strings.Trim(sp[1], " ")

		fmt.Printf("arg env: %s:%s\n", name, value)
		b.argEnv[name] = value

	}
	return nil
}
