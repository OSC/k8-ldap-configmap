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

	"github.com/OSC/k8-ldap-configmap/internal/config"
	ldapgo "github.com/go-ldap/ldap/v3"
	"github.com/prometheus/common/promslog"
)

func TestGetUserHomes(t *testing.T) {
	// Create a mock LDAP search result with user entries
	users := &ldapgo.SearchResult{
		Entries: []*ldapgo.Entry{
			{
				DN: "uid=testuser1,ou=people,dc=example,dc=com",
				Attributes: []*ldapgo.EntryAttribute{
					{
						Name:   "uid",
						Values: []string{"testuser1"},
					},
					{
						Name:   "homeDirectory",
						Values: []string{"/home/testuser1"},
					},
				},
			},
			{
				DN: "uid=testuser2,ou=people,dc=example,dc=com",
				Attributes: []*ldapgo.EntryAttribute{
					{
						Name:   "uid",
						Values: []string{"testuser2"},
					},
					{
						Name:   "homeDirectory",
						Values: []string{"/home/testuser2"},
					},
				},
			},
		},
	}

	// Create a mock config
	mockConfig := &config.Config{
		UserPrefix: "user-",
		UserAttrMap: map[string]string{
			"name": "uid",
			"home": "homeDirectory",
		},
	}

	// Create the mapper
	mapper := NewUserHomeMapper(mockConfig, promslog.NewNopLogger())

	// Get the data
	data, err := mapper.GetData(users, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the results
	if len(data) != 2 {
		t.Errorf("Unexpected length of data, got: %d", len(data))
	}

	// Check first user
	if val, ok := data["user-testuser1"]; !ok {
		t.Errorf("user-testuser1 not found in data")
	} else if val != "/home/testuser1" {
		t.Errorf("Unexpected value for user-testuser1, got: %s", val)
	}

	// Check second user
	if val, ok := data["user-testuser2"]; !ok {
		t.Errorf("user-testuser2 not found in data")
	} else if val != "/home/testuser2" {
		t.Errorf("Unexpected value for user-testuser2, got: %s", val)
	}
}
