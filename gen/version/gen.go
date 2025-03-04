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
	"github.com/cuisongliu/automq-operator/defaults"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s IMAGE_NAME\n", os.Args[0])
		os.Exit(1)
	}
	imageName := os.Args[1]
	fmt.Printf("image name is %s", imageName)
	_ = os.MkdirAll("deploy/images/shim", 0755)
	_ = os.WriteFile("deploy/images/shim/image.txt", []byte(defaults.DefaultImageName), 0755)
	_ = os.WriteFile("deploy/images/shim/busybox.txt", []byte(defaults.BusyboxImageName), 0755)
	cmdUpgradeImageName := fmt.Sprintf("sed -i '/#replace_by_makefile/!b;n;c\\image: %s' deploy/charts/automq-operator/values.yaml", imageName)
	if err := execCmd("bash", "-c", cmdUpgradeImageName); err != nil {
		fmt.Printf("execCmd error %v", err)
		os.Exit(1)
	}
	version := strings.Split(imageName, ":")
	if len(version) != 2 {
		fmt.Printf("image name error")
		os.Exit(1)
	}
	shotVersion := strings.ReplaceAll(version[1], "v", "")
	if shotVersion == "latest" {
		shotVersion = "0.0.0"
	}
	cmdUpgradeChartVersion := fmt.Sprintf("sed -i '/#replace_by_makefile/!b;n;c\\version: %s' deploy/charts/automq-operator/Chart.yaml", shotVersion)
	if err := execCmd("bash", "-c", cmdUpgradeChartVersion); err != nil {
		fmt.Printf("execCmd error %v", err)
		os.Exit(1)
	}

	cmdUpgradeReadme := fmt.Sprintf("scripts/release_tag.sh v%s", shotVersion)
	if err := execCmd("bash", "-c", cmdUpgradeReadme); err != nil {
		fmt.Printf("execCmd error %v", err)
		os.Exit(1)
	}

	cmdUpgradeReadmeSealos := fmt.Sprintf("scripts/release_tag_sealos.sh v%s", shotVersion)
	if err := execCmd("bash", "-c", cmdUpgradeReadmeSealos); err != nil {
		fmt.Printf("execCmd error %v", err)
		os.Exit(1)
	}

	fmt.Printf("update image success")
}

func execCmd(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}
