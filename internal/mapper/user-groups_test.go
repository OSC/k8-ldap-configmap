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
	"testing"
	"time"

	"github.com/OSC/k8-ldap-configmap/internal/config"
	"github.com/OSC/k8-ldap-configmap/internal/ldap"
	"github.com/OSC/k8-ldap-configmap/internal/test"
	"github.com/go-kit/kit/log"
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

func TestGetDataMemberOf(t *testing.T) {
	mapper := NewUserGroupsMapper(_config, log.NewNopLogger())
	l, err := ldap.LDAPConnect(_config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	users, err := ldap.LDAPUsers(l, _config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	groups, err := ldap.LDAPGroups(l, _config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	data, err := mapper.GetData(users, groups)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 4 {
		t.Errorf("Unexpected length of data, got: %d", len(data))
	}
	if val, ok := data["testuser1"]; !ok {
		t.Errorf("testuser1 not found in data")
	} else if val != "[\"testgroup1\",\"testgroup2\"]" {
		t.Errorf("Unexpected value, got:%s", val)
	}
}

func TestGetDataMember(t *testing.T) {
	_config.MemberScheme = "member"
	mapper := NewUserGroupsMapper(_config, log.NewNopLogger())
	l, err := ldap.LDAPConnect(_config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	users, err := ldap.LDAPUsers(l, _config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	groups, err := ldap.LDAPGroups(l, _config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	data, err := mapper.GetData(users, groups)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 4 {
		t.Errorf("Unexpected length of data, got: %d", len(data))
	}
	if val, ok := data["testuser1"]; !ok {
		t.Errorf("testuser1 not found in data")
	} else if val != "[\"testgroup1\"]" {
		t.Errorf("Unexpected value for testuser1, got:%s", val)
	}
	if val, ok := data["testuser2"]; !ok {
		t.Errorf("testuser2 not found in data")
	} else if val != "[\"testgroup1\",\"testgroup2\"]" {
		t.Errorf("Unexpected value for testuser2, got:%s", val)
	}
}

func TestGetDataMemberUID(t *testing.T) {
	_config.MemberScheme = "memberuid"
	mapper := NewUserGroupsMapper(_config, log.NewNopLogger())
	l, err := ldap.LDAPConnect(_config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	users, err := ldap.LDAPUsers(l, _config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	groups, err := ldap.LDAPGroups(l, _config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	data, err := mapper.GetData(users, groups)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 4 {
		t.Errorf("Unexpected length of data, got: %d", len(data))
	}
	if val, ok := data["testuser1"]; !ok {
		t.Errorf("testuser1 not found in data")
	} else if val != "[\"testgroup1\"]" {
		t.Errorf("Unexpected value for testuser1, got:%s", val)
	}
	if val, ok := data["testuser2"]; !ok {
		t.Errorf("testuser2 not found in data")
	} else if val != "[\"testgroup1\",\"testgroup2\"]" {
		t.Errorf("Unexpected value for testuser2, got:%s", val)
	}
}
