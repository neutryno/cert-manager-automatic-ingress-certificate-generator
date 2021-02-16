package main

import (
    "github.com/stretchr/testify/assert"
    v1 "k8s.io/api/networking/v1"
    "os"
    "testing"
)

func clearEnvs() {
    var envsToClear []string
    envsToClear = append(envsToClear, "CLUSTER_ISSUERS")
    envsToClear = append(envsToClear, "CLUSTER_ISSUER_1_REGEX")
    envsToClear = append(envsToClear, "CLUSTER_ISSUER_2_REGEX")

    for _, i := range envsToClear {
        os.Unsetenv(i)
    }
}

func TestLookupEnv(t *testing.T) {
    clearEnvs()
    env_key := "CLUSTER_ISSUERS"
    env_value := "dev-cloud-issuer,test-cloud-issuer"
    os.Setenv(env_key, env_value)
    issuernames := LookupEnv(env_key)

    assert.Equal(t, env_value, issuernames)
}

func TestLookupEnvThatDoesNotExist(t *testing.T) {
    clearEnvs()
    assert.Panics(t, func() { LookupEnv("notexistentenv")}, "Should panic.")
}

func TestGetClusterIssuers(t *testing.T) {
    clearEnvs()
    env_ci_key := "CLUSTER_ISSUERS"
    env_ci_value := "dev-cloud-issuer,test-cloud-issuer"
    os.Setenv(env_ci_key, env_ci_value)
    env_regex1_key := "CLUSTER_ISSUER_1_REGEX"
    env_regex1_value := "(.*)\\.dev\\.cloud\\.domain\\.de"
    os.Setenv(env_regex1_key, env_regex1_value)
    env_regex2_key := "CLUSTER_ISSUER_2_REGEX"
    env_regex2_value := "(.*)\\.test\\.cloud\\.domain\\.de"
    os.Setenv(env_regex2_key, env_regex2_value)

    issuers := getClusterIssuers()

    assert.Equal(t, "dev-cloud-issuer", issuers[0].name)
    assert.Equal(t, "(.*)\\.dev\\.cloud\\.domain\\.de", issuers[0].regex)
    assert.Equal(t, "test-cloud-issuer", issuers[1].name)
    assert.Equal(t, "(.*)\\.test\\.cloud\\.domain\\.de", issuers[1].regex)
}

func TestGetClusterIssuersShouldPanicIfRegexForSecoundClusterIssuerDoesNotExist(t *testing.T) {
    clearEnvs()
    env_ci_key := "CLUSTER_ISSUERS"
    env_ci_value := "dev-cloud-issuer,test-cloud-issuer"
    os.Setenv(env_ci_key, env_ci_value)
    env_regex1_key := "CLUSTER_ISSUER_1_REGEX"
    env_regex1_value := "(.*)\\.dev\\.cloud\\.domain\\.de"
    os.Setenv(env_regex1_key, env_regex1_value)

    assert.Panics(t, func() {getClusterIssuers() })
}

func TestIncludesHost(t *testing.T) {
    clearEnvs()
    tls := v1.IngressTLS{Hosts: []string{"hello.dev.cloud.domain.de"}}
    assert.True(t, includesHost([]v1.IngressTLS{tls}, "hello.dev.cloud.domain.de"))

    tls = v1.IngressTLS{Hosts: []string{"hello.dev.cloud.domain.de"}}
    assert.False(t, includesHost([]v1.IngressTLS{tls}, "ciao.dev.cloud.domain.de"))
}

func TestIncludesCertManagerAnnotation(t *testing.T) {
    ingress := v1.Ingress{ }
    ingress.SetAnnotations(map[string]string{"cert-manager.io/cluster-issuer": "dev-cloud-issuer"})

    issuer := ClusterIssuer{name: "dev-cloud-issuer", regex: ".*"}
    assert.True(t, includesCertManagerAnnotation(ingress, issuer))
}

func TestIncludesCertManagerAnnotationNotExists(t *testing.T) {
    ingress := v1.Ingress{ }
    ingress.SetAnnotations(map[string]string{"other-annotation": "dev-cloud-issuer"})

    issuer := ClusterIssuer{name: "dev-cloud-issuer", regex: ".*"}
    assert.False(t, includesCertManagerAnnotation(ingress, issuer))
}

func TestIncludesCertManagerAnnotationOtherIssuer(t *testing.T) {
    ingress := v1.Ingress{ }
    ingress.SetAnnotations(map[string]string{"cert-manager.io/cluster-issuer": "test-cloud-issuer"})

    issuer := ClusterIssuer{name: "dev-cloud-issuer", regex: ".*"}
    assert.False(t, includesCertManagerAnnotation(ingress, issuer))
}