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
	registerMapper("user-home", []string{"name", "home"}, nil, NewUserHomeMapper)
}

func NewUserHomeMapper(config *config.Config, logger *slog.Logger) Mapper {
	return &UserHome{
		config: config,
		logger: logger,
	}
}

type UserHome struct {
	config *config.Config
	logger *slog.Logger
}

func (m UserHome) Name() string {
	return "user-home"
}

func (m UserHome) ConfigMapName() string {
	return "user-home-map"
}

func (m UserHome) GetData(users *ldap.SearchResult, groups *ldap.SearchResult) (map[string]string, error) {
	m.logger.Debug("Mapper running")
	userHomes := make(map[string]string)
	for _, entry := range users.Entries {
		name := fmt.Sprintf("%s%s", m.config.UserPrefix, entry.GetAttributeValue(m.config.UserAttrMap["name"]))
		home := entry.GetAttributeValue(m.config.UserAttrMap["home"])
		userHomes[name] = home
	}
	m.logger.Debug("Mapper complete", "user-homes", len(userHomes))
	return userHomes, nil
}
