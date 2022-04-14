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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
)

var (
	errorEnvVariableNameEmpty      = errors.New("env variable empty or not set")
	errorUnsupportedArguments      = errors.New("argument not supported")
	errorInvalidEnvArgument        = errors.New("invalid env passed via argument")
	errorEnvVariableNameNotAllowed = errors.New("env variable not allowed")
	errorInvalidFilename           = errors.New("invalid filename")
	errorEmptyFilename             = errors.New("filename is not set")
	errorInvalidRegistry           = errors.New("invalid registry")
)

var dockerRegistry = "docker.io"

type KoBuild struct {
	ko   string
	args []string
	envs map[string]string
}

func KoBuildNew(ko string) *KoBuild {
	c := KoBuild{
		ko:   ko,
		envs: make(map[string]string),
		args: make([]string, 0),
	}

	return &c
}

func (b *KoBuild) Run(dry bool) error {
	fmt.Println("Run")

	command, err := b.generateCommandArgs()
	if err != nil {
		return err
	}

	envs, err := b.generateEnvVariables()
	if err != nil {
		return err
	}

	registry, err := b.generateRegistry()
	if err != nil {
		return err
	}

	// A dry run prints the information that is "trusted", before
	// the compiler is invoked.
	if dry {
		// Share the command.
		command, err := marshallList(command)
		if err != nil {
			return err
		}
		fmt.Printf("::set-output name=command::%s\n", command)

		// Share the env variables.
		env, err := b.generateCommandEnvVariables()
		if err != nil {
			return err
		}
		envs, err := marshallList(env)
		if err != nil {
			return err
		}
		fmt.Printf("::set-output name=envs::%s\n", envs)

		fmt.Printf("::set-output name=registry::%s\n", registry)

		return nil
	}

	fmt.Println("command", command)
	fmt.Println("env", envs)
	fmt.Println("registry", registry)
	return syscall.Exec(b.ko, command, envs)
}

func (b *KoBuild) SetArgs(args string) error {
	if args == "" {
		return nil
	}

	for _, arg := range strings.Split(args, " ") {
		arg = strings.Trim(arg, " ")

		fmt.Printf("arg: %s\n", arg)
		b.args = append(b.args, arg)

	}
	return nil
}

func (b *KoBuild) generateEnvVariables() ([]string, error) {
	env := os.Environ()

	cenv, err := b.generateCommandEnvVariables()
	if err != nil {
		return cenv, err
	}

	env = append(env, cenv...)

	return env, nil
}

func (b *KoBuild) generateCommandEnvVariables() ([]string, error) {
	var env []string

	// Set env variables from config file.
	for k, v := range b.envs {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env, nil
}

func (b *KoBuild) generateRegistry() (string, error) {
	var registry string
	for k, v := range b.envs {
		if k == "KO_DOCKER_REPO" {
			registry = v
		}
	}

	// Empty registry is allowed, default to docker.
	if registry == "" {
		return dockerRegistry, nil
	}

	parts := strings.Split(registry, "/")
	if len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", errorInvalidRegistry, registry)
	}

	// A non-separated string indicates a docker username
	// https://github.com/google/ko#choose-destination.
	if len(parts) == 1 {
		return dockerRegistry, nil
	}

	return strings.Trim(parts[0], " "), nil
}

func marshallList(args []string) (string, error) {
	jsonData, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(jsonData)
	if err != nil {
		return "", fmt.Errorf("base64.StdEncoding.DecodeString: %w", err)
	}
	return encoded, nil
}

func (b *KoBuild) generateCommandArgs() ([]string, error) {
	flags := []string{b.ko, "publish"}

	for _, v := range b.args {
		flags = append(flags, v)
	}
	return flags, nil
}

func (b *KoBuild) SetArgEnvVariables(envs string) error {
	if envs == "" {
		return nil
	}

	for _, e := range strings.Split(envs, ",") {
		s := strings.Trim(e, " ")

		sp := strings.Split(s, "=")
		if len(sp) != 2 {
			return fmt.Errorf("%w: %s", errorInvalidEnvArgument, s)
		}
		name := strings.Trim(sp[0], " ")
		value := strings.Trim(sp[1], " ")

		fmt.Printf("arg env: %s:%s\n", name, value)
		b.envs[name] = value

	}
	return nil
}
