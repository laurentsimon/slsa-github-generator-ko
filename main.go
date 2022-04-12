// Copyright SLSA team.
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

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/laurentsimon/slsa-github-generator-ko/builder/pkg"
)

func usage(p string) {
	panic(fmt.Sprintf(`Usage: 
	 %s build [--dry] --env $ENV
	 %s predicate --binary-name $NAME --digest $DIGEST --command $COMMAND --env $ENV`, p, p))
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	// Build command.
	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
	buildDry := buildCmd.Bool("dry", false, "dry run of the build without invoking compiler")
	buildEnv := buildCmd.String("env", "", "env variables used to compile the binary")

	// Predicate command.
	predicateCmd := flag.NewFlagSet("predicate", flag.ExitOnError)
	predicateName := predicateCmd.String("binary-name", "", "untrusted binary name of the artifact built")
	predicateDigest := predicateCmd.String("digest", "", "sha256 digest of the untrusted binary")
	predicateCommand := predicateCmd.String("command", "", "command used to compile the binary")
	// predicateEnv := predicateCmd.String("env", "", "env variables used to compile the binary")

	// Expect a sub-command.
	if len(os.Args) < 2 {
		usage(os.Args[0])
	}

	switch os.Args[1] {
	case buildCmd.Name():
		buildCmd.Parse(os.Args[2:])
		if len(buildCmd.Args()) < 1 {
			usage(os.Args[0])
		}

		ko, err := exec.LookPath("ko")
		check(err)

		kobuild := pkg.KoBuildNew(ko)

		// Set env variables encoded as arguments.
		err = kobuild.SetArgEnvVariables(*buildEnv)
		check(err)

		err = kobuild.Run(*buildDry)
		check(err)
	case predicateCmd.Name():
		predicateCmd.Parse(os.Args[2:])
		// Note: *predicateEnv may be empty.
		if *predicateName == "" || *predicateDigest == "" ||
			*predicateCommand == "" {
			usage(os.Args[0])
		}

		/*githubContext, ok := os.LookupEnv("GITHUB_CONTEXT")
		if !ok {
			panic(errors.New("environment variable GITHUB_CONTEXT not present"))
		}

		attBytes, err := pkg.GeneratePredicate(*predicateName, *predicateDigest,
			githubContext, *predicateCommand, *predicateEnv)
		check(err)

		filename := fmt.Sprintf("%s.intoto.jsonl", *predicateName)
		err = ioutil.WriteFile(filename, attBytes, 0600)
		check(err)

		fmt.Printf("::set-output name=signed-predicate-name::%s\n", filename)
		*/
	default:
		fmt.Println("expected 'build' or 'predicate' subcommands")
		os.Exit(1)
	}
}
