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
	"encoding/json"
	"log/slog"
	"sort"

	"github.com/OSC/k8-ldap-configmap/internal/config"
	ldap "github.com/go-ldap/ldap/v3"
)

func init() {
	registerMapper("user-groups", []string{"name", "gid"}, []string{"name", "gid"}, NewUserGroupsMapper)
}

func NewUserGroupsMapper(config *config.Config, logger *slog.Logger) Mapper {
	return &UserGroups{
		config: config,
		logger: logger,
	}
}

type UserGroups struct {
	config *config.Config
	logger *slog.Logger
}

func (m UserGroups) Name() string {
	return "user-groups"
}

func (m UserGroups) ConfigMapName() string {
	return "user-groups-map"
}

func (m UserGroups) GetData(users *ldap.SearchResult, groups *ldap.SearchResult) (map[string]string, error) {
	m.logger.Debug("Mapper running")
	data, err := GetUserGroups(users, groups, m.config, m.logger)
	if err != nil {
		return nil, err
	}
	userGroups := make(map[string]string)
	for user, groups := range data {
		groupNames := []string{}
		for _, group := range groups {
			groupNames = append(groupNames, group.name)
		}
		sort.Strings(groupNames)
		userGroupsJSON, _ := json.Marshal(groupNames)
		userGroups[user] = string(userGroupsJSON)
	}
	m.logger.Debug("Mapper complete", "user-groups", len(userGroups))
	return userGroups, nil
}
