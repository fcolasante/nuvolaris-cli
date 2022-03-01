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
	_ "embed"
	"fmt"
	"os/exec"

	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type KindConfig struct {
	homedir              string
	kindYaml             []byte
	nuvolarisClusterName string
	nuvolarisConfigDir   string
	kindConfigFile       string
	fullConfigPath       string
	preflightChecks      func(string) error
	kind                 func(...string) error
}

//go:embed embed/kind.yaml
var kind_yaml []byte

var img string = "openwhisk/action-nodejs-v14"
var tag string = ":nightly"

func configKind() (*KindConfig, error) {

	homeDir, err := GetHomeDir()
	if err != nil {
		return nil, err
	}

	config := KindConfig{
		homedir:              homeDir,
		kindYaml:             kind_yaml,
		nuvolarisClusterName: "nuvolaris",
		nuvolarisConfigDir:   ".nuvolaris",
		kindConfigFile:       "kind.yaml",
		fullConfigPath:       "",
		preflightChecks:      RunPreflightChecks,
		kind:                 Kind,
	}
	return &config, nil
}

func (config *KindConfig) manageKindCluster(action string) error {

	switch action {
	case "create":
		if err := config.createCluster(); err != nil {
			return err
		}
	case "destroy":
		if err := config.destroyCluster(); err != nil {
			return err
		}
	default:
		fmt.Println("did you mean nuv devcluster create/destroy?")
	}
	return nil
}

func (config *KindConfig) createCluster() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error in create cluster: %w", err)
		}
	}()

	clusterIsRunning, err := config.clusterAlreadyRunning()
	if err != nil {
		return err
	}
	if clusterIsRunning {
		fmt.Println("nuvolaris kind cluster is already running...skipping")
		return nil
	}

	fmt.Println("running preflight checks")
	if err = config.preflightChecks(config.homedir); err != nil {
		return err
	}
	fmt.Println("preflight checks ok")

	err = config.createNuvolarisConfigDirIfNotExists()
	if err != nil {
		return err
	}

	fullConfigPath, err := config.rewriteKindConfigFile()
	if err != nil {
		return err
	}

	config.fullConfigPath = fullConfigPath

	fmt.Println("starting nuvolaris kind cluster...hang tight")
	if err = config.startCluster(); err != nil {
		return err
	}

	fmt.Println("preloading openwhisk docker image...")
	if err = config.preloadOpenWhiskImage(); err != nil {
		return err
	}

	fmt.Println("nuvolaris kind cluster started. Have a nice day! 👋")
	return nil
}

func (config *KindConfig) destroyCluster() error {
	clusterIsRunning, err := config.clusterAlreadyRunning()
	if err != nil {
		return err
	}
	if clusterIsRunning {
		if err := config.stopCluster(); err != nil {
			return err
		}
		fmt.Println("kind cluster nuvolaris destroyed")
	} else {
		fmt.Println("kind cluster nuvolaris not found...skipping")
	}
	return nil
}

func (config *KindConfig) clusterAlreadyRunning() (bool, error) {
	//capture cmd output
	rescue_stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := config.kind("get", "clusters")

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescue_stdout

	if err != nil {
		return false, err
	}
	if strings.Contains(string(out), config.nuvolarisClusterName) {
		return true, nil
	} else {
		return false, nil
	}
}

func (config *KindConfig) createNuvolarisConfigDirIfNotExists() error {
	fullPath := filepath.Join(config.homedir, config.nuvolarisConfigDir)
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		if err := os.Mkdir(fullPath, 0777); err != nil {
			return err
		}
		fmt.Println("nuvolaris config dir created")
	}
	return nil
}

func (config *KindConfig) rewriteKindConfigFile() (string, error) {
	path := filepath.Join(config.homedir, config.nuvolarisConfigDir, config.kindConfigFile)
	if _, err := os.Stat(path); err == nil {
		os.Remove(path)
	}
	if err := os.WriteFile(path, config.kindYaml, 0600); err != nil {
		return "", err
	}
	fmt.Println(config.kindConfigFile + " written")
	return path, nil
}

func (config *KindConfig) startCluster() error {
	if err := config.kind("create", "cluster", "--wait=1m", "--config="+config.fullConfigPath); err != nil {
		return err
	}
	return nil

}

func (config *KindConfig) preloadOpenWhiskImage() error {
	if !isOpenWhiskImageLoaded() {
		exec.Command("docker", "pull", img)
	}
	if err := config.kind("load", "docker-image", img+tag, "--name="+config.nuvolarisClusterName); err != nil {
		return err
	}
	return nil
}

func isOpenWhiskImageLoaded() bool {
	cmd := exec.Command("docker", "images", "-q", img+tag)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	if string(out) == "" {
		return false
	} else {
		return true
	}
}

func (config *KindConfig) stopCluster() error {
	if err := config.kind("delete", "cluster", "--name="+config.nuvolarisClusterName); err != nil {
		return err
	}
	return nil
}
