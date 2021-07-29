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
	"testing"

	"github.com/OSC/k8-ldap-configmap/internal/ldap"
	"github.com/go-kit/kit/log"
)

func TestGetUserGIDsDataMemberOf(t *testing.T) {
	_config.MemberScheme = "memberof"
	mapper := NewUserGIDsMapper(_config, log.NewNopLogger())
	l, err := ldap.LDAPConnect(_config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	users, err := ldap.LDAPUsers(l, _config.UserFilter, _config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	groups, err := ldap.LDAPGroups(l, _config.GroupFilter, _config, log.NewNopLogger())
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
	} else if val != "[\"1000\",\"1001\"]" {
		t.Errorf("Unexpected value, got:%s", val)
	}
}

func TestGetUserGIDsDataMember(t *testing.T) {
	_config.MemberScheme = "member"
	mapper := NewUserGIDsMapper(_config, log.NewNopLogger())
	l, err := ldap.LDAPConnect(_config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	users, err := ldap.LDAPUsers(l, _config.UserFilter, _config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	groups, err := ldap.LDAPGroups(l, _config.GroupFilter, _config, log.NewNopLogger())
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
	} else if val != "[\"1001\"]" {
		t.Errorf("Unexpected value for testuser1, got:%s", val)
	}
	if val, ok := data["testuser2"]; !ok {
		t.Errorf("testuser2 not found in data")
	} else if val != "[\"1000\",\"1001\"]" {
		t.Errorf("Unexpected value for testuser2, got:%s", val)
	}
}

func TestGetUserGIDsDataMemberUID(t *testing.T) {
	_config.MemberScheme = "memberuid"
	mapper := NewUserGIDsMapper(_config, log.NewNopLogger())
	l, err := ldap.LDAPConnect(_config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	users, err := ldap.LDAPUsers(l, _config.UserFilter, _config, log.NewNopLogger())
	if err != nil {
		t.Fatal(err)
	}
	groups, err := ldap.LDAPGroups(l, _config.GroupFilter, _config, log.NewNopLogger())
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
	} else if val != "[\"1001\"]" {
		t.Errorf("Unexpected value for testuser1, got:%s", val)
	}
	if val, ok := data["testuser2"]; !ok {
		t.Errorf("testuser2 not found in data")
	} else if val != "[\"1000\",\"1001\"]" {
		t.Errorf("Unexpected value for testuser2, got:%s", val)
	}
}
