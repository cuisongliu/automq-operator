/*
Copyright 2024 cuisongliu@qq.com.

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

package controller

import (
	"strings"

	"github.com/cuisongliu/automq-operator/defaults"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func sysctlContainer() v1.Container {
	container := v1.Container{
		Name:            "sysctl",
		Image:           defaults.BusyboxImageName,
		Command:         nil,
		ImagePullPolicy: "IfNotPresent",
		SecurityContext: &v1.SecurityContext{
			Privileged: &[]bool{true}[0],
		},
		Resources: v1.ResourceRequirements{Limits: map[v1.ResourceName]resource.Quantity{
			v1.ResourceCPU:    resource.MustParse("500m"),
			v1.ResourceMemory: resource.MustParse("256Mi"),
		}, Requests: map[v1.ResourceName]resource.Quantity{
			v1.ResourceCPU:    resource.MustParse("10m"),
			v1.ResourceMemory: resource.MustParse("64Mi"),
		}},
	}

	container.Command = []string{
		"sh",
		"-c",
		strings.Join(sysctls, "\n"),
	}
	return container
}

var sysctls = []string{
	"sysctl -w fs.inotify.max_user_watches=8000000",
	"sysctl -w fs.file-max=40265318",
	"sysctl -w fs.inotify.max_user_instances=12800",
	"sysctl -w fs.inotify.max_queued_events=8000000",
	"sysctl -w net.core.somaxconn=65535",
	"sysctl -w net.ipv4.ip_local_port_range=\"1024 65535\"",
	"sysctl -w net.ipv4.tcp_tw_reuse=1",
	"sysctl -w net.ipv4.tcp_fin_timeout=10",
	"sysctl -w net.ipv4.tcp_keepalive_intvl=75",
	"sysctl -w net.ipv4.tcp_keepalive_probes=9",
	"sysctl -w net.ipv4.tcp_keepalive_time=7200",
	"ulimit -a",
	"mkdir -p /etc/security",
	"echo \"* - nofile 1048576\" >> /etc/security/limits.conf",
	"echo \"* - nproc 1048576\" >> /etc/security/limits.conf",
	"echo \"root - nofile 1048576\" >> /etc/security/limits.conf",
	"echo \"root - nproc 1048576\" >> /etc/security/limits.conf",
}
