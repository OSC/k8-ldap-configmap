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

package test

import (
	"fmt"
	"log"
	"os"

	"github.com/lor00x/goldap/message"
	ldap "github.com/vjeantet/ldapserver"
)

const (
	BindDN      = "cn=test,dc=test"
	GroupBaseDN = "ou=Groups,dc=test"
	UserBaseDN  = "ou=People,dc=test"
	GroupFilter = "(objectClass=posixGroup)"
	UserFilter  = "(objectClass=posixAccount)"
)

func LdapServer() *ldap.Server {
	ldap.Logger = log.New(os.Stdout, "[server] ", log.LstdFlags)
	server := ldap.NewServer()
	routes := ldap.NewRouteMux()
	routes.NotFound(handleNotFound)
	routes.Bind(handleBind)
	routes.Search(handleSearchGroup).
		BaseDn(GroupBaseDN).
		Filter(GroupFilter).
		Label("SEARCH - GROUP")
	routes.Search(handleSearchUser).
		BaseDn(UserBaseDN).
		Filter(UserFilter).
		Label("SEARCH - USER")
	routes.Search(handleSearch).Label("SEARCH - NO MATCH")
	server.Handle(routes)
	return server
}

func handleBind(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetBindRequest()
	res := ldap.NewBindResponse(ldap.LDAPResultSuccess)
	if r.AuthenticationChoice() == "simple" {
		if string(r.Name()) != BindDN {
			res.SetResultCode(ldap.LDAPResultInvalidCredentials)
			res.SetDiagnosticMessage("invalid credentials")
		}
	}
	w.Write(res)
}

func handleNotFound(w ldap.ResponseWriter, r *ldap.Message) {
	switch r.ProtocolOpType() {
	case ldap.ApplicationBindRequest:
		res := ldap.NewBindResponse(ldap.LDAPResultSuccess)
		res.SetDiagnosticMessage("Default binding behavior set to return Success")

		w.Write(res)

	default:
		res := ldap.NewResponse(ldap.LDAPResultUnwillingToPerform)
		res.SetDiagnosticMessage("Operation not implemented by server")
		w.Write(res)
	}
}

func handleSearchGroup(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetSearchRequest()
	data := map[string]map[string][]string{
		"testgroup1": {
			"objectClass": []string{"posixGroup"},
			"gidNumber":   []string{"1000"},
			"memberUid":   []string{"testuser1"},
			"member": []string{
				fmt.Sprintf("cn=testuser1,%s", GroupBaseDN),
			},
		},
		"testgroup2": {
			"objectClass": []string{"posixGroup"},
			"gidNumber":   []string{"1001"},
			"memberUid":   []string{"testuser2", "testuser3"},
			"member": []string{
				fmt.Sprintf("cn=testuser2,%s", GroupBaseDN),
				fmt.Sprintf("cn=testuser3,%s", GroupBaseDN),
			},
		},
	}
	for cn, attrs := range data {
		dn := fmt.Sprintf("cn=%s,%s", cn, r.BaseObject())
		e := ldap.NewSearchResultEntry(dn)
		e.AddAttribute("cn", message.AttributeValue(cn))
		for key, value := range attrs {
			values := []message.AttributeValue{}
			for _, v := range value {
				values = append(values, message.AttributeValue(v))
			}
			e.AddAttribute(message.AttributeDescription(key), values...)
		}
		w.Write(e)
	}
	res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultSuccess)
	w.Write(res)
}

func handleSearchUser(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetSearchRequest()
	data := map[string]map[string][]string{
		"testuser1": {
			"objectClass": []string{"posixAccount"},
			"uidNumber":   []string{"1000"},
			"gidNumber":   []string{"1000"},
			"memberOf": []string{
				fmt.Sprintf("cn=testgroup1,%s", UserBaseDN),
				fmt.Sprintf("cn=testgroup2,%s", UserBaseDN),
			},
		},
		"testuser2": {
			"objectClass": []string{"posixAccount"},
			"uidNumber":   []string{"1001"},
			"gidNumber":   []string{"1000"},
			"memberOf": []string{
				fmt.Sprintf("cn=testgroup2,%s", UserBaseDN),
			},
		},
		"testuser3": {
			"objectClass": []string{"posixAccount"},
			"uidNumber":   []string{"1002"},
			"gidNumber":   []string{"1001"},
			"memberOf": []string{
				fmt.Sprintf("cn=testgroup2,%s", UserBaseDN),
			},
		},
		"testuser4": {
			"objectClass": []string{"posixAccount"},
			"uidNumber":   []string{"1003"},
			"gidNumber":   []string{"1001"},
		},
	}
	for cn, attrs := range data {
		dn := fmt.Sprintf("cn=%s,%s", cn, r.BaseObject())
		e := ldap.NewSearchResultEntry(dn)
		e.AddAttribute("cn", message.AttributeValue(cn))
		e.AddAttribute("uid", message.AttributeValue(cn))
		for key, value := range attrs {
			values := []message.AttributeValue{}
			for _, v := range value {
				values = append(values, message.AttributeValue(v))
			}
			e.AddAttribute(message.AttributeDescription(key), values...)
		}
		w.Write(e)
	}
	res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultSuccess)
	w.Write(res)
}

func handleSearch(w ldap.ResponseWriter, m *ldap.Message) {
	res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultNoSuchObject)
	w.Write(res)
}
