// Copyright 2020 Ohio Supercomputer Center
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/OSC/k8-ldap-configmap/internal/mapper"
	"github.com/OSC/k8-ldap-configmap/internal/metrics"
	"github.com/OSC/k8-ldap-configmap/internal/test"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus/testutil"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	ldapserver = "127.0.0.1:10389"
)

var (
	baseArgs = []string{
		fmt.Sprintf("--ldap-url=ldap://%s", ldapserver),
		fmt.Sprintf("--ldap-group-base-dn=%s", test.GroupBaseDN),
		fmt.Sprintf("--ldap-user-base-dn=%s", test.UserBaseDN),
		fmt.Sprintf("--ldap-bind-dn=%s", test.BindDN),
		"--ldap-bind-password=password",
		"--namespace=test",
	}
)

func TestMain(m *testing.M) {
	if _, err := kingpin.CommandLine.Parse(baseArgs); err != nil {
		os.Exit(1)
	}

	server := test.LdapServer()
	go func() {
		err := server.ListenAndServe(ldapserver)
		if err != nil {
			os.Exit(1)
		}
	}()
	time.Sleep(1 * time.Second)

	exitVal := m.Run()
	server.Stop()
	os.Exit(exitVal)
}

func clientset() kubernetes.Interface {
	clientset := fake.NewSimpleClientset(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}, &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user-uid",
			Namespace: "test",
		},
	})
	return clientset
}

func TestRun(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse(baseArgs); err != nil {
		t.Fatal(err)
	}
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)

	resetCounters()
	clientset := clientset()
	config := createConfig()
	mappers := mapper.GetMappers(config, logger)
	err := run(mappers, config, clientset, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	userUIDMap, err := clientset.CoreV1().ConfigMaps("test").Get(context.TODO(), "user-uid-map", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting configmap: %v", err)
	}
	if len(userUIDMap.Data) != 4 {
		t.Errorf("Unexpected number of items in configmap data")
	}
	if val, ok := userUIDMap.Data["testuser2"]; !ok {
		t.Errorf("Configmap is missing testuser2")
	} else if val != "1001" {
		t.Errorf("Configmap value for testuser2 is incorrect")
	}
	userGIDMap, err := clientset.CoreV1().ConfigMaps("test").Get(context.TODO(), "user-gid-map", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting configmap: %v", err)
	}
	if len(userGIDMap.Data) != 4 {
		t.Errorf("Unexpected number of items in configmap data")
	}
	if val, ok := userGIDMap.Data["testuser2"]; !ok {
		t.Errorf("Configmap is missing testuser2")
	} else if val != "1000" {
		t.Errorf("Configmap value for testuser2 is incorrect")
	}

	expected := `
	# HELP k8_ldap_configmap_error Indicates an error was encountered
	# TYPE k8_ldap_configmap_error gauge
	k8_ldap_configmap_error 0
	# HELP k8_ldap_configmap_errors_total Total number of errors
	# TYPE k8_ldap_configmap_errors_total counter
	k8_ldap_configmap_errors_total{mapper="user-gid"} 0
	k8_ldap_configmap_errors_total{mapper="user-uid"} 0
	# HELP k8_ldap_configmap_keys_count Number of data keys in ConfigMap
	# TYPE k8_ldap_configmap_keys_count gauge
	k8_ldap_configmap_keys_count{configmap="user-gid-map"} 4
	k8_ldap_configmap_keys_count{configmap="user-uid-map"} 4
	# HELP k8_ldap_configmap_size_bytes Size of ConfigMap in bytes
	# TYPE k8_ldap_configmap_size_bytes gauge
	k8_ldap_configmap_size_bytes{configmap="user-gid-map"} 202
	k8_ldap_configmap_size_bytes{configmap="user-uid-map"} 202
	`

	if err := testutil.GatherAndCompare(metrics.MetricGathers(false), strings.NewReader(expected),
		"k8_ldap_configmap_error", "k8_ldap_configmap_errors_total",
		"k8_ldap_configmap_size_bytes", "k8_ldap_configmap_keys_count"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestRunGroups(t *testing.T) {
	args := []string{
		"--mappers=user-groups",
	}
	args = append(args, baseArgs...)
	if _, err := kingpin.CommandLine.Parse(args); err != nil {
		t.Fatal(err)
	}
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)

	resetCounters()
	clientset := clientset()
	config := createConfig()
	mappers := mapper.GetMappers(config, logger)
	err := run(mappers, config, clientset, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	userGroupsMap, err := clientset.CoreV1().ConfigMaps("test").Get(context.TODO(), "user-groups-map", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting configmap: %v", err)
		return
	}
	if len(userGroupsMap.Data) != 4 {
		t.Errorf("Unexpected number of items in configmap data")
	}
	if val, ok := userGroupsMap.Data["testuser1"]; !ok {
		t.Errorf("Configmap is missing testuser1")
	} else if val != "[\"testgroup1\",\"testgroup2\"]" {
		t.Errorf("Configmap value for testuser1 is incorrect: %s", val)
	}
	if val, ok := userGroupsMap.Data["testuser2"]; !ok {
		t.Errorf("Configmap is missing testuser2")
	} else if val != "[\"testgroup1\",\"testgroup2\"]" {
		t.Errorf("Configmap value for testuser2 is incorrect: %s", val)
	}
	if val, ok := userGroupsMap.Data["testuser3"]; !ok {
		t.Errorf("Configmap is missing testuser3")
	} else if val != "[\"testgroup2\"]" {
		t.Errorf("Configmap value for testuser3 is incorrect: %s", val)
	}

	expected := `
	# HELP k8_ldap_configmap_error Indicates an error was encountered
	# TYPE k8_ldap_configmap_error gauge
	k8_ldap_configmap_error 0
	# HELP k8_ldap_configmap_errors_total Total number of errors
	# TYPE k8_ldap_configmap_errors_total counter
	k8_ldap_configmap_errors_total{mapper="user-groups"} 0
	# HELP k8_ldap_configmap_keys_count Number of data keys in ConfigMap
	# TYPE k8_ldap_configmap_keys_count gauge
	k8_ldap_configmap_keys_count{configmap="user-groups-map"} 4
	# HELP k8_ldap_configmap_size_bytes Size of ConfigMap in bytes
	# TYPE k8_ldap_configmap_size_bytes gauge
	k8_ldap_configmap_size_bytes{configmap="user-groups-map"} 283
	`

	if err := testutil.GatherAndCompare(metrics.MetricGathers(false), strings.NewReader(expected),
		"k8_ldap_configmap_error", "k8_ldap_configmap_errors_total",
		"k8_ldap_configmap_size_bytes", "k8_ldap_configmap_keys_count"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func resetCounters() {
	metrics.MetricErrorsTotal.Reset()
	metrics.MetricConfigMapSize.Reset()
	metrics.MetricConfigMapKeys.Reset()
}

func TestValidateArgs(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err == nil {
		t.Errorf("Expected error parsing lack of args")
	}
	args := []string{
		"--ldap-url=ldap://ldap:389",
		fmt.Sprintf("--ldap-group-base-dn=%s", test.GroupBaseDN),
		fmt.Sprintf("--ldap-user-base-dn=%s", test.UserBaseDN),
		"--namespace=test",
	}
	if _, err := kingpin.CommandLine.Parse(args); err != nil {
		t.Errorf("Error parsing args %s", err.Error())
	}
	err := validateArgs(log.NewNopLogger())
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	args = append(args, []string{
		fmt.Sprintf("--ldap-bind-dn=%s", test.BindDN),
		"--ldap-bind-password=",
		"--ldap-user-attr-map=name=uid",
		"--ldap-group-attr-map=name=cn",
		"--ldap-member-scheme=foo",
		"--mappers=user-uid,user-gid,user-groups,foobar",
	}...)
	if _, err := kingpin.CommandLine.Parse(args); err != nil {
		t.Errorf("Error parsing args %s", err.Error())
	}
	err = validateArgs(log.NewNopLogger())
	if err == nil {
		t.Errorf("Expected errors")
	}
	if !strings.Contains(err.Error(), "ldap-user-attr-map") {
		t.Errorf("Expected error about missing uid key")
	}
	if !strings.Contains(err.Error(), "ldap-user-attr-map") {
		t.Errorf("Expected error about missing gid key")
	}
	if !strings.Contains(err.Error(), "ldap-group-attr-map") {
		t.Errorf("Expected error about missing gid key")
	}
	if !strings.Contains(err.Error(), "mappers") {
		t.Errorf("Expected error about invalid mapper")
	}
	if !strings.Contains(err.Error(), "ldap-member-scheme") {
		t.Errorf("Expected error about invalid member scheme")
	}
	if !strings.Contains(err.Error(), "ldap-bind") {
		t.Errorf("Expected error about missing bind args")
	}
}

func TestSetupLogging(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	for _, l := range levels {
		args := []string{fmt.Sprintf("--log-level=%s", l)}
		args = append(baseArgs, args...)
		if _, err := kingpin.CommandLine.Parse(args); err != nil {
			t.Fatal(err)
		}
		logger := setupLogging()
		if logger == nil {
			t.Errorf("Unexpected error getting logger")
		}
	}
	args := []string{"--log-format=json"}
	args = append(baseArgs, args...)
	if _, err := kingpin.CommandLine.Parse(args); err != nil {
		t.Fatal(err)
	}
	logger := setupLogging()
	if logger == nil {
		t.Errorf("Unexpected error getting logger")
	}
	args = []string{"--log-level=foo"}
	args = append(baseArgs, args...)
	if _, err := kingpin.CommandLine.Parse(args); err != nil {
		t.Fatal(err)
	}
	logger = setupLogging()
	if logger != nil {
		t.Errorf("Expected an error getting logger")
	}
}
