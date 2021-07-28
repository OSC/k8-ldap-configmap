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

package mapper

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/OSC/k8-ldap-configmap/internal/config"
	"github.com/OSC/k8-ldap-configmap/internal/test"
)

const (
	ldapserver = "127.0.0.1:10390"
)

var (
	_config = &config.Config{
		LdapURL:     fmt.Sprintf("ldap://%s", ldapserver),
		GroupBaseDN: test.GroupBaseDN,
		UserBaseDN:  test.UserBaseDN,
		BindDN:      test.BindDN,
		GroupFilter: test.GroupFilter,
		UserFilter:  test.UserFilter,
		GroupAttrMap: map[string]string{
			"name": "cn",
			"gid":  "gidNumber",
		},
		UserAttrMap: map[string]string{
			"name": "uid",
			"uid":  "uidNumber",
			"gid":  "gidNumber",
		},
		MemberScheme: "memberof",
	}
)

func TestMain(m *testing.M) {
	server := test.LdapServer()
	go func() {
		err := server.ListenAndServe(ldapserver)
		if err != nil {
			os.Exit(1)
		}
	}()
	time.Sleep(1 * time.Second)

	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestInitRequiredUserAttrs(t *testing.T) {
	if val, ok := requiredUserAttrs["user-gid"]; !ok {
		t.Errorf("user user-gid key missing")
	} else if !reflect.DeepEqual(val, []string{"name", "gid"}) {
		t.Errorf("unexpected required attrs for user user-gid, got %v", val)
	}
	if val, ok := requiredUserAttrs["user-uid"]; !ok {
		t.Errorf("user user-uid key missing")
	} else if !reflect.DeepEqual(val, []string{"name", "uid"}) {
		t.Errorf("unexpected required attrs for user user-uid, got %v", val)
	}
	if val, ok := requiredUserAttrs["user-groups"]; !ok {
		t.Errorf("user user-groups key missing")
	} else if !reflect.DeepEqual(val, []string{"name", "gid"}) {
		t.Errorf("unexpected required attrs for user user-groups, got %v", val)
	}
	if val, ok := requiredUserAttrs["user-gids"]; !ok {
		t.Errorf("user user-gids key missing")
	} else if !reflect.DeepEqual(val, []string{"name", "gid"}) {
		t.Errorf("unexpected required attrs for user user-gids, got %v", val)
	}
}

func TestInitRequiredGroupAttrs(t *testing.T) {
	if val, ok := requiredGroupAttrs["user-gid"]; !ok {
		t.Errorf("group user-gid key missing")
	} else if val != nil {
		t.Errorf("unexpected required attrs for group user-gid, got %v", val)
	}
	if val, ok := requiredGroupAttrs["user-uid"]; !ok {
		t.Errorf("group user-uid key missing")
	} else if val != nil {
		t.Errorf("unexpected required attrs for group user-uid, got %v", val)
	}
	if val, ok := requiredGroupAttrs["user-groups"]; !ok {
		t.Errorf("group user-groups key missing")
	} else if !reflect.DeepEqual(val, []string{"name", "gid"}) {
		t.Errorf("unexpected required attrs for group user-groups, got %v", val)
	}
	if val, ok := requiredGroupAttrs["user-gids"]; !ok {
		t.Errorf("group user-gids key missing")
	} else if !reflect.DeepEqual(val, []string{"name", "gid"}) {
		t.Errorf("unexpected required attrs for group user-gids, got %v", val)
	}
}

func TestValidMappers(t *testing.T) {
	expected := []string{"user-gid", "user-groups", "user-uid", "user-gids"}
	value := ValidMappers()
	sort.Strings(value)
	sort.Strings(expected)
	if !reflect.DeepEqual(value, expected) {
		t.Errorf("Unexpected value for valid mappers\nExpected: %v\nGot: %v", expected, value)
	}
}

func TestParseDN(t *testing.T) {
	value := ParseDN("cn=test,dc=test,dc=com")
	if value != "test" {
		t.Errorf("Unexpected value from parsing DN\nExpected: test\nGot: %s", value)
	}
	value = ParseDN("")
	if value != "" {
		t.Errorf("Unexpected value from parsing DN\nExpected: <empty string>\nGot: %s", value)
	}
	value = ParseDN("test,dc=test,dc=com")
	if value != "" {
		t.Errorf("Unexpected value from parsing DN\nExpected: <empty string>\nGot: %s", value)
	}
}

func TestRequiredAttrs(t *testing.T) {
	expected := []string{"name", "uid", "gid"}
	value := RequiredAttrs("user", []string{"user-gid", "user-uid"})
	sort.Strings(value)
	sort.Strings(expected)
	if !reflect.DeepEqual(value, expected) {
		t.Errorf("Unexpected value for user required attrs\nExpected: %v\nGot: %v", expected, value)
	}
	expected = []string{}
	value = RequiredAttrs("group", []string{"user-gid", "user-uid"})
	if !reflect.DeepEqual(value, expected) {
		t.Errorf("Unexpected value for group required attrs when disabled\nGot: %v", value)
	}
	expected = []string{"name", "gid"}
	value = RequiredAttrs("group", []string{"user-gid", "user-uid", "user-groups"})
	sort.Strings(value)
	sort.Strings(expected)
	if !reflect.DeepEqual(value, expected) {
		t.Errorf("Unexpected value for group required attrs\nExpected: %v\nGot: %v", expected, value)
	}
}
