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

	"github.com/OSC/k8-ldap-configmap/internal/config"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	ldap "github.com/go-ldap/ldap/v3"
)

func init() {
	registerMapper("user-gids", []string{"name", "gid"}, []string{"name", "gid"}, NewUserGIDsMapper)
}

func NewUserGIDsMapper(config *config.Config, logger log.Logger) Mapper {
	return &UserGIDs{
		config: config,
		logger: logger,
	}
}

type UserGIDs struct {
	config *config.Config
	logger log.Logger
}

func (m UserGIDs) Name() string {
	return "user-gids"
}

func (m UserGIDs) ConfigMapName() string {
	return "user-gids-map"
}

func (m UserGIDs) GetData(users *ldap.SearchResult, groups *ldap.SearchResult) (map[string]string, error) {
	level.Debug(m.logger).Log("msg", "Mapper running")
	data, err := GetUserGroups(users, groups, m.config)
	if err != nil {
		return nil, err
	}
	userGIDs := make(map[string]string)
	for user, groups := range data {
		groupGIDs := []string{}
		for _, group := range groups {
			groupGIDs = append(groupGIDs, group.gid)
		}
		groupGIDsJSON, _ := json.Marshal(groupGIDs)
		userGIDs[user] = string(groupGIDsJSON)
	}
	level.Debug(m.logger).Log("msg", "Mapper complete", "user-gids", len(userGIDs))
	return userGIDs, nil
}
