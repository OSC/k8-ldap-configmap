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

	"github.com/OSC/k8-ldap-configmap/internal/config"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	ldap "github.com/go-ldap/ldap/v3"
)

func init() {
	registerMapper("user-uid", []string{"name", "uid"}, nil, NewUserUIDMapper)
}

func NewUserUIDMapper(config *config.Config, logger log.Logger) Mapper {
	return &UserUID{
		config: config,
		logger: logger,
	}
}

type UserUID struct {
	config *config.Config
	logger log.Logger
}

func (m UserUID) Name() string {
	return "user-uid"
}

func (m UserUID) ConfigMapName() string {
	return "user-uid-map"
}

func (m UserUID) GetData(users *ldap.SearchResult, groups *ldap.SearchResult) (map[string]string, error) {
	level.Debug(m.logger).Log("msg", "Mapper running")
	userUIDs := make(map[string]string)
	for _, entry := range users.Entries {
		name := fmt.Sprintf("%s%s", m.config.UserPrefix, entry.GetAttributeValue(m.config.UserAttrMap["name"]))
		uid := entry.GetAttributeValue(m.config.UserAttrMap["uid"])
		userUIDs[name] = uid
	}
	level.Debug(m.logger).Log("msg", "Mapper complete", "user-uids", len(userUIDs))
	return userUIDs, nil
}
