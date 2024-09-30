/*
Copyright 2023 cuisongliu@qq.com.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) != 1 {
		fmt.Printf("Usage: %s IMAGE_NAME\n", os.Args[0])
		os.Exit(1)
	}
	imageName := os.Args[1]
	fmt.Printf("image name is %s", imageName)
	_ = os.MkdirAll("deploy/images/shim", 0755)
	_ = os.WriteFile("deploy/images/shim/image.txt", []byte(imageName), 0755)
	cmd1 := fmt.Sprintf("sed -i '/#replace_by_makefile/!b;n;c\\image: %s' deploy/charts/automq-operator/values.yaml", imageName)
	if err := execCmd("bash", "-c", cmd1); err != nil {
		fmt.Printf("execCmd error %v", err)
		os.Exit(1)
	}
	version := strings.Split(imageName, ":")
	if len(version) != 2 {
		fmt.Printf("image name error")
		os.Exit(1)
	}
	shotVersion := strings.ReplaceAll(version[1], "v", "")
	cmd2 := fmt.Sprintf("sed -i '/#replace_by_makefile/!b;n;c\\version: %s' deploy/charts/automq-operator/Chart.yaml", shotVersion)
	if err := execCmd("bash", "-c", cmd2); err != nil {
		fmt.Printf("execCmd error %v", err)
		os.Exit(1)
	}
}

func execCmd(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}
