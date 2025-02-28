// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
package main

import (
	"fmt"
	"time"
)

type SetupPipeline struct {
	kubeClient          *KubeClient
	k8sContext          string
	operatorDockerImage string
	err                 error
	logger              *Logger
}

type setupStep func(sp *SetupPipeline)

func (sp *SetupPipeline) step(f setupStep) {
	if sp.err != nil {
		return
	}
	f(sp)
	time.Sleep(2 * time.Second)
}

func setupNuvolaris(logger *Logger, cmd *SetupCmd) error {

	setupWithNoFlags := !cmd.Devcluster &&
		!cmd.Configure &&
		cmd.ImageTag == ImageTag &&
		cmd.Uninstall == "" &&
		cmd.Context == ""

	if setupWithNoFlags {
		err := listAvailableContexts()
		if err != nil {
			return err
		}
		return nil
	}

	if cmd.ImageTag != ImageTag && cmd.Context == "" {
		fmt.Println("Specify Kubernetes context with --context flag")
		return nil
	}

	if cmd.Configure {
		//TODO setup ~/.nuvolaris/config.yaml
	}

	imgTag := cmd.ImageTag
	sp := SetupPipeline{
		operatorDockerImage: "ghcr.io/nuvolaris/nuvolaris-operator:" + imgTag,
		logger:              logger,
	}

	if cmd.Devcluster {
		sp.err = startDevCluster(sp.logger)
		sp.k8sContext = "kind-nuvolaris"
	}

	if cmd.Context != "" {
		sp.k8sContext = cmd.Context
	}

	if cmd.Uninstall != "" {
		sp.k8sContext = cmd.Uninstall
		sp.kubeClient, sp.err = initClients(sp.k8sContext)
		sp.step(resetNuvolaris)
		return sp.err
	} else {
		sp.kubeClient, sp.err = initClients(sp.k8sContext)
		sp.step(createNuvolarisNamespace)
		sp.step(deployServiceAccount)
		sp.step(deployClusterRoleBinding)
		sp.step(runNuvolarisOperatorPod)
		sp.step(deployOperatorObject)
		sp.step(waitForOpenWhiskReady)
		return sp.err
	}
}

func createNuvolarisNamespace(sp *SetupPipeline) {
	sp.err = sp.kubeClient.createNuvolarisNamespace()
}

func deployServiceAccount(sp *SetupPipeline) {
	sp.err = sp.kubeClient.createServiceAccount()
}

func deployClusterRoleBinding(sp *SetupPipeline) {
	sp.err = sp.kubeClient.createClusterRoleBinding()
}

func runNuvolarisOperatorPod(sp *SetupPipeline) {
	sp.err = sp.kubeClient.createOperatorPod(sp.operatorDockerImage)
}

func deployOperatorObject(sp *SetupPipeline) {
	sp.err = createWhiskOperatorObject(sp.kubeClient)
}

func waitForOpenWhiskReady(sp *SetupPipeline) {
	sp.err = readinessProbe(sp.kubeClient)
}

func resetNuvolaris(sp *SetupPipeline) {
	sp.err = sp.kubeClient.cleanup()
}
