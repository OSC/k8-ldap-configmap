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
	"log/slog"

	"github.com/OSC/k8-ldap-configmap/internal/config"
	ldap "github.com/go-ldap/ldap/v3"
)

func init() {
	registerMapper("user-gid", []string{"name", "gid"}, nil, NewUserGIDMapper)
}

func NewUserGIDMapper(config *config.Config, logger *slog.Logger) Mapper {
	return &UserGID{
		config: config,
		logger: logger,
	}
}

type UserGID struct {
	config *config.Config
	logger *slog.Logger
}

func (m UserGID) Name() string {
	return "user-gid"
}

func (m UserGID) ConfigMapName() string {
	return "user-gid-map"
}

func (m UserGID) GetData(users *ldap.SearchResult, groups *ldap.SearchResult) (map[string]string, error) {
	m.logger.Debug("Mapper running")
	userGIDs := make(map[string]string)
	for _, entry := range users.Entries {
		name := fmt.Sprintf("%s%s", m.config.UserPrefix, entry.GetAttributeValue(m.config.UserAttrMap["name"]))
		uid := entry.GetAttributeValue(m.config.UserAttrMap["gid"])
		userGIDs[name] = uid
	}
	m.logger.Debug("Mapper complete", "user-gids", len(userGIDs))
	return userGIDs, nil
}
