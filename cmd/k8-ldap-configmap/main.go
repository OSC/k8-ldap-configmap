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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/OSC/k8-ldap-configmap/internal/config"
	localldap "github.com/OSC/k8-ldap-configmap/internal/ldap"
	"github.com/OSC/k8-ldap-configmap/internal/mapper"
	"github.com/OSC/k8-ldap-configmap/internal/metrics"
	"github.com/OSC/k8-ldap-configmap/internal/utils"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	ldap "github.com/go-ldap/ldap/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"golang.org/x/sync/errgroup"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	appName     = "k8-ldap-configmap"
	metricsPath = "/metrics"
)

var (
	ldapURL               = kingpin.Flag("ldap-url", "LDAP URL").Envar("LDAP_URL").Required().String()
	ldapTLS               = kingpin.Flag("ldap-tls", "Enable TLS connection to LDAP server").Envar("LDAP_TLS").Default("false").Bool()
	ldapTLSVerify         = kingpin.Flag("ldap-tls-verify", "Verify TLS certificate with LDAP server").Envar("LDAP_TLS_VERIFY").Default("true").Bool()
	ldapTLSCACert         = kingpin.Flag("ldap-tls-ca-cert", "TLS CA Cert for LDAP server").Envar("LDAP_TLS_CA_CERT").String()
	ldapGroupBaseDN       = kingpin.Flag("ldap-group-base-dn", "LDAP Group Base DN").Envar("LDAP_GROUP_BASE_DN").Required().String()
	ldapUserBaseDN        = kingpin.Flag("ldap-user-base-dn", "LDAP User Base DN").Envar("LDAP_USER_BASE_DN").Required().String()
	ldapBindDN            = kingpin.Flag("ldap-bind-dn", "LDAP Bind DN").Envar("LDAP_BIND_DN").String()
	ldapBindPassword      = kingpin.Flag("ldap-bind-password", "LDAP Bind Password").Envar("LDAP_BIND_PASSWORD").String()
	ldapGroupFilter       = kingpin.Flag("ldap-group-filter", "LDAP group filter").Default("(objectClass=posixGroup)").Envar("LDAP_GROUP_FILTER").String()
	ldapUserFilter        = kingpin.Flag("ldap-user-filter", "LDAP user filter").Default("(objectClass=posixAccount)").Envar("LDAP_USER_FILTER").String()
	ldapPagedSearch       = kingpin.Flag("ldap-paged-search", "Enable LDAP paged searching").Default("false").Envar("LDAP_PAGED_SEARCH").Bool()
	ldapPagedSearchSize   = kingpin.Flag("ldap-paged-search-size", " LDAP paged search size").Default("1000").Envar("LDAP_PAGED_SEARCH_SIZE").Int()
	ldapMemberScheme      = kingpin.Flag("ldap-member-scheme", "Scheme used to define group members, either memberof, member or memberuid").Default("memberof").Envar("LDAP_MEMBER_SCHEME").String()
	ldapUserAttrMap       = kingpin.Flag("ldap-user-attr-map", "Attribute map for users").Default(config.DefaultUserAttrMap).Envar("LDAP_USER_ATTR_MAP").String()
	ldapGroupAttrMap      = kingpin.Flag("ldap-group-attr-map", "Attribute map for groups").Default(config.DefaultGroupAttrMap).Envar("LDAP_GROUP_ATTR_MAP").String()
	mappersArg            = kingpin.Flag("mappers", "Comma separated list of mappers to generate.").Default("user-uid,user-gid").Envar("MAPPERS").String()
	namespace             = kingpin.Flag("namespace", "namespace for ConfigMaps").Envar("NAMESPACE").Required().String()
	userPrefix            = kingpin.Flag("user-prefix", "Prefix to add to user names").Envar("USER_PREFIX").String()
	interval              = kingpin.Flag("interval", "Duration between sync runs").Default("5m").Envar("INTERLVAL").Duration()
	listenAddress         = kingpin.Flag("listen-address", "Address to listen for HTTP requests").Default(":8080").Envar("LISTEN_ADDRESS").String()
	processMetrics        = kingpin.Flag("process-metrics", "Collect metrics about running process such as CPU and memory and Go stats").Default("true").Envar("PROCESS_METRICS").Bool()
	kubeconfig            = kingpin.Flag("kubeconfig", "Path to kubeconfig when running outside Kubernetes cluster").Default("").Envar("KUBECONFIG").String()
	logLevel              = kingpin.Flag("log-level", "Log level, One of: [debug, info, warn, error]").Default("info").Envar("LOG_LEVEL").String()
	logFormat             = kingpin.Flag("log-format", "Log format, One of: [logfmt, json]").Default("logfmt").Envar("LOG_FORMAT").String()
	validLdapMemberScheme = []string{"memberof", "member", "memberuid"}
	timestampFormat       = log.TimestampFormat(
		func() time.Time { return time.Now().UTC() },
		"2006-01-02T15:04:05.000Z07:00",
	)
)

func main() {
	kingpin.Version(version.Print(appName))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := setupLogging()
	if logger == nil {
		os.Exit(1)
	}

	var err error
	err = validateArgs(logger)
	if err != nil {
		os.Exit(1)
	}

	var config *rest.Config
	if *kubeconfig == "" {
		level.Info(logger).Log("msg", "Loading in cluster kubeconfig", "kubeconfig", *kubeconfig)
		config, err = rest.InClusterConfig()
	} else {
		level.Info(logger).Log("msg", "Loading kubeconfig", "kubeconfig", *kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	}
	if err != nil {
		level.Error(logger).Log("msg", "Error loading kubeconfig", "err", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		level.Error(logger).Log("msg", "Unable to generate Clientset", "err", err)
		os.Exit(1)
	}

	c := createConfig()
	mappers := mapper.GetMappers(c, logger)

	level.Info(logger).Log("msg", fmt.Sprintf("Starting %s", appName), "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
	             <head><title>` + appName + `</title></head>
	             <body>
	             <h1>` + appName + `</h1>
	             <p><a href='` + metricsPath + `'>Metrics</a></p>
	             </body>
	             </html>`))
	})
	http.Handle(metricsPath, promhttp.HandlerFor(metrics.MetricGathers(*processMetrics), promhttp.HandlerOpts{}))

	go func() {
		if err := http.ListenAndServe(*listenAddress, nil); err != nil {
			level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
			os.Exit(1)
		}
	}()

	for {
		var errNum float64
		err = run(mappers, c, clientset, logger)
		if err != nil {
			errNum = 1
		}
		metrics.MetricError.Set(errNum)
		level.Debug(logger).Log("msg", "Sleeping for interval", "interval", fmt.Sprintf("%.0f", (*interval).Seconds()))
		time.Sleep(*interval)
	}
}

func run(mappers []mapper.Mapper, config *config.Config, clientset kubernetes.Interface, logger log.Logger) error {
	start := time.Now()
	metrics.MetricLastRun.Set(float64(start.Unix()))
	defer metrics.MetricDuration.Set(time.Since(start).Seconds())
	l, err := localldap.LDAPConnect(config, logger)
	if err != nil {
		return err
	}
	defer l.Close()

	var groupResults, userResults *ldap.SearchResult
	var groupErr, userErr error
	searchWG := &sync.WaitGroup{}
	searchWG.Add(2)
	go func() {
		defer searchWG.Done()
		if len(config.RequiredGroupAttrs) == 0 {
			return
		}
		groupResults, groupErr = localldap.LDAPGroups(l, config, logger)
	}()
	go func() {
		defer searchWG.Done()
		if len(config.RequiredUserAttrs) == 0 {
			return
		}
		userResults, userErr = localldap.LDAPUsers(l, config, logger)
	}()
	searchWG.Wait()
	if groupErr != nil {
		return groupErr
	}
	if userErr != nil {
		return userErr
	}

	errs, _ := errgroup.WithContext(context.Background())
	for _, m := range mappers {
		_m := m
		errs.Go(func() error {
			data, err := _m.GetData(userResults, groupResults)
			if err != nil {
				metrics.MetricErrorsTotal.WithLabelValues(_m.Name()).Inc()
				return err
			}
			err = configmap(clientset, _m.ConfigMapName(), data, logger)
			if err != nil {
				metrics.MetricErrorsTotal.WithLabelValues(_m.Name()).Inc()
				return err
			}
			return nil
		})
	}
	return errs.Wait()
}

func configmap(clientset kubernetes.Interface, name string, data map[string]string, logger log.Logger) error {
	var err error
	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: *namespace,
		},
		Data: data,
	}
	var action string
	if _, err = clientset.CoreV1().ConfigMaps(*namespace).Get(context.TODO(), name, metav1.GetOptions{}); k8errors.IsNotFound(err) {
		action = "create"
		_, err = clientset.CoreV1().ConfigMaps(*namespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
	} else {
		action = "update"
		_, err = clientset.CoreV1().ConfigMaps(*namespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
	}
	if err == nil {
		level.Info(logger).Log("msg", "ConfigMap sync successful", "action", action, "name", name, "namespace", *namespace)
		metrics.MetricConfigMapKeys.WithLabelValues(name).Set(float64(len(data)))
		configMapJSON, err := json.Marshal(configMap)
		if err != nil {
			level.Error(logger).Log("msg", "Unable to marshall configmap to JSON", "name", name, "namespace", *namespace, "err", err)
			return err
		}
		metrics.MetricConfigMapSize.WithLabelValues(name).Set(float64(len(configMapJSON)))
	} else {
		level.Error(logger).Log("msg", "Failed to sync ConfigMap", "action", action, "name", name, "namespace", *namespace, "err", err)
	}
	return err
}

func createConfig() *config.Config {
	userAttrMap := utils.AttrMap(*ldapUserAttrMap)
	groupAttrMap := utils.AttrMap(*ldapGroupAttrMap)
	enabledMappers := strings.Split(*mappersArg, ",")
	requiredUserAttrs := mapper.RequiredAttrs("user", enabledMappers)
	requiredGroupAttrs := mapper.RequiredAttrs("group", enabledMappers)
	return &config.Config{
		LdapURL:            *ldapURL,
		LdapTLS:            *ldapTLS,
		LdapTLSVerify:      *ldapTLSVerify,
		LdapTLSCACert:      *ldapTLSCACert,
		BindDN:             *ldapBindDN,
		BindPassword:       *ldapBindPassword,
		UserBaseDN:         *ldapUserBaseDN,
		GroupBaseDN:        *ldapGroupBaseDN,
		UserFilter:         *ldapUserFilter,
		GroupFilter:        *ldapGroupFilter,
		UserAttrMap:        userAttrMap,
		GroupAttrMap:       groupAttrMap,
		RequiredUserAttrs:  requiredUserAttrs,
		RequiredGroupAttrs: requiredGroupAttrs,
		PagedSearch:        *ldapPagedSearch,
		PagedSearchSize:    *ldapPagedSearchSize,
		MemberScheme:       *ldapMemberScheme,
		UserPrefix:         *userPrefix,
		EnabledMappers:     enabledMappers,
	}
}

func validateArgs(logger log.Logger) error {
	errs := []string{}
	validMappers := mapper.ValidMappers()
	enabledMappers := strings.Split(*mappersArg, ",")
	userAttrs := mapper.RequiredAttrs("user", enabledMappers)
	groupAttrs := mapper.RequiredAttrs("group", enabledMappers)
	var err error
	if len(groupAttrs) > 0 {
		groupAttrMap := utils.AttrMap(*ldapGroupAttrMap)
		groupAttrMapKeys := utils.MapKeysStrings(groupAttrMap)
		for _, attr := range groupAttrs {
			if !utils.SliceContains(groupAttrMapKeys, attr) {
				errs = append(errs, fmt.Sprintf("ldap-group-attr-map=\"Missing group attribute map key '%s'\"", attr))
			}
		}
	}
	if len(userAttrs) > 0 {
		userAttrMap := utils.AttrMap(*ldapUserAttrMap)
		userAttrMapKeys := utils.MapKeysStrings(userAttrMap)
		for _, attr := range userAttrs {
			if !utils.SliceContains(userAttrMapKeys, attr) {
				errs = append(errs, fmt.Sprintf("ldap-user-attr-map=\"Missing user attribute map key '%s'\"", attr))
			}
		}
	}
	if (*ldapBindDN != "" && *ldapBindPassword == "") || (*ldapBindDN == "" && *ldapBindPassword != "") {
		errs = append(errs, "ldap-bind=\"Must provide both LDAP Bind DN and Bind Password if either is provided\"")
	}
	if !utils.SliceContains(validLdapMemberScheme, *ldapMemberScheme) {
		errs = append(errs, fmt.Sprintf("ldap-member-scheme=\"LDAP member scheme '%s' invalid\"", *ldapMemberScheme))
	}
	for _, mapper := range strings.Split(*mappersArg, ",") {
		if !utils.SliceContains(validMappers, mapper) {
			errs = append(errs, fmt.Sprintf("mappers=\"Defined mapper %s is not valid\"", mapper))
		}
	}
	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, ", "))
		level.Error(logger).Log("err", err)
	}
	return err
}

func setupLogging() log.Logger {
	var logger log.Logger
	if *logFormat == "json" {
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	} else {
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	}
	switch *logLevel {
	case "debug":
		logger = level.NewFilter(logger, level.AllowDebug())
	case "info":
		logger = level.NewFilter(logger, level.AllowInfo())
	case "warn":
		logger = level.NewFilter(logger, level.AllowWarn())
	case "error":
		logger = level.NewFilter(logger, level.AllowError())
	default:
		logger = level.NewFilter(logger, level.AllowError())
		level.Error(logger).Log("msg", "Unrecognized log level", "level", *logLevel)
		return nil
	}
	logger = log.With(logger, "ts", timestampFormat, "caller", log.DefaultCaller)
	return logger
}
