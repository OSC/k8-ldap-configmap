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

package utils

import (
	"reflect"
	"sort"
	"testing"

	"github.com/OSC/k8-ldap-configmap/internal/config"
)

func TestSliceContains(t *testing.T) {
	value := SliceContains([]string{"foo", "bar"}, "bar")
	if value != true {
		t.Errorf("Expected slice to contain value")
	}
	value = SliceContains([]string{"foo"}, "bar")
	if value != false {
		t.Errorf("Expected slice not to contain value")
	}
}

func TestMapKeysStrings(t *testing.T) {
	input := map[string]string{
		"foo": "bar",
		"baz": "faz",
	}
	expected := []string{"foo", "baz"}
	value := MapKeysStrings(input)
	sort.Strings(expected)
	sort.Strings(value)
	if !reflect.DeepEqual(value, expected) {
		t.Errorf("Unexpected value for map keys\nExpected: %v\nGot: %v", expected, value)
	}
}

func TestUserAttrMapDefaults(t *testing.T) {
	expected := map[string]string{
		"name": "uid",
		"uid":  "uidNumber",
		"gid":  "gidNumber",
	}
	value := AttrMap(config.DefaultUserAttrMap)
	if !reflect.DeepEqual(value, expected) {
		t.Errorf("Unexpected value for user attr map\nExpected: %v\nGot: %v", expected, value)
	}
}

func TestGroupAttrMapDefaults(t *testing.T) {
	expected := map[string]string{
		"name": "cn",
		"gid":  "gidNumber",
	}
	value := AttrMap(config.DefaultGroupAttrMap)
	if !reflect.DeepEqual(value, expected) {
		t.Errorf("Unexpected value for user attr map\nExpected: %v\nGot: %v", expected, value)
	}
}

func TestAttrMapErrors(t *testing.T) {
	expected := map[string]string{
		"name": "uid",
	}
	value := AttrMap("name=uid,foo")
	if !reflect.DeepEqual(value, expected) {
		t.Errorf("Unexpected value for user attr map\nExpected: %v\nGot: %v", expected, value)
	}
}
