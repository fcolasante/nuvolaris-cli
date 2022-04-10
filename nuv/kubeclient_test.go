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
	"context"
	_ "embed"
	"strings"
	"testing"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclient "k8s.io/client-go/kubernetes/fake"
)

var testclient KubeClient = KubeClient{
	namespace: "nuvolaris",
	ctx:       context.Background(),
}

var nspace = &coreV1.Namespace{
	ObjectMeta: metaV1.ObjectMeta{
		Name: testclient.namespace,
	},
}

func TestCreateNamespace(t *testing.T) {
	testclient.clientset = fakeclient.NewSimpleClientset()
	// given namespace does not exist yet
	_, err := testclient.clientset.CoreV1().Namespaces().Get(testclient.ctx, testclient.namespace, metaV1.GetOptions{})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf(err.Error())
	}
	// when namespace is created
	err = testclient.createNuvolarisNamespace()
	if err != nil {
		t.Errorf(err.Error())
	}
	// then namespace will exist
	_, err = testclient.clientset.CoreV1().Namespaces().Get(testclient.ctx, testclient.namespace, metaV1.GetOptions{})
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestNamespaceNotCreatedIfAlreadyExists(t *testing.T) {
	// given namespace already exists
	testclient.clientset = fakeclient.NewSimpleClientset(nspace)

	// when we try to create namespace
	err := testclient.createNuvolarisNamespace()

	// then nothing should happen
	if err != nil {
		t.Errorf(err.Error())
	}
}

func Example_createNamespace() {
	testclient.clientset = fakeclient.NewSimpleClientset()
	testclient.createNuvolarisNamespace()
	//Output:
	//✓ Namespace nuvolaris created
}

func Example_namespaceIsNotCreatedIfAlreadyExists() {
	testclient.clientset = fakeclient.NewSimpleClientset(nspace)
	testclient.createNuvolarisNamespace()
	//Output:
	//namespace nuvolaris already exists...skipping
}
