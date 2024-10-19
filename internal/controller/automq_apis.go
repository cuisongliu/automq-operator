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
	"context"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func APIRegistry(ctx context.Context, k8sClient client.Client) {
	setupLog := ctrl.Log.WithName("setup")
	setupLog.Info("cache sync success")
	router := gin.Default()
	router.GET("/api/v1/nodes/:name", func(c *gin.Context) {
		name := c.Param("name")
		node := &v1.Node{}
		node.Name = name
		if noe := k8sClient.Get(ctx, client.ObjectKeyFromObject(node), node); noe != nil {
			c.JSON(500, gin.H{"message": noe.Error()})
			return
		}
		nodeIP := ""
		for _, addr := range node.Status.Addresses {
			if addr.Type == v1.NodeInternalIP {
				nodeIP = addr.Address
				break
			}
		}
		if nodeIP == "" {
			c.JSON(500, gin.H{"message": "node ip not found"})
			return
		}
		c.String(200, nodeIP)
	})
	router.Run(":9090")
}
