package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ClusterIssuer struct {
	name  string
	regex string
}

func main() {

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the client
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	clusterIssuers := getClusterIssuers()

	log.Info("cert-manager-automatic-ingress-certificate-generator started")
	for {

		// get all ingresses in all namespaces
		ingresses, err := client.NetworkingV1().Ingresses("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}

		// for each ingress
		for _, in := range ingresses.Items {
			log.WithField("ingress", in.Name).WithField("namesapce", in.Namespace).
				Debug("Processing Ingress")
			// for each rule in spec
			for _, rules := range in.Spec.Rules {
				// for each cluster issuer
				for _, issuer := range clusterIssuers {
					var re = regexp.MustCompile(issuer.regex)

					if len(re.FindStringIndex(rules.Host)) > 0 {
						// ingress rule contains host matching to cluster issuer regex
						log.WithField("ingress", in.Name).WithField("namesapce", in.Namespace).
							WithField("host", rules.Host).
							WithField("cluster issuer", issuer.name).
							WithField("cluster issuer regex", issuer.regex).
							Debug("Found matching host.")

						if includesHost(in.Spec.TLS, rules.Host) && includesCertManagerAnnotation(in, issuer) {
							log.WithField("ingress", in.Name).
								WithField("namesapce", in.Namespace).
								WithField("tls", in.Spec.TLS).
								Debug("Ingress has TLS and Annotation")
						} else {
							// Update Annotation section of Ingress
							if !includesCertManagerAnnotation(in, issuer) {
								in.Annotations["cert-manager.io/cluster-issuer"] = issuer.name
								result, err := client.NetworkingV1().Ingresses(in.Namespace).
									Update(context.TODO(), &in, metav1.UpdateOptions{})
								if err != nil {
									log.WithField("ingress", in.Name).WithField("namesapce", in.Namespace).
										WithField("cluster issuer", issuer.name).
										Error("Unable to update Annotations: ", err.Error())
								} else {
									// successful update of annotation
									log.WithField("ingress", result.Name).
										WithField("namesapce", in.Namespace).
										WithField("annotation", result.Annotations).
										Info("Ingress Annotations updated")
								}
							}

							// Update TLS section of Ingress
							if !includesHost(in.Spec.TLS, rules.Host) {
								host := rules.Host
								in.Spec.TLS = append(in.Spec.TLS, v1.IngressTLS{
									Hosts: []string {host},
									SecretName: strings.ReplaceAll(host, ".", "-") + "-crt",
								})
								result, err := client.NetworkingV1().Ingresses(in.Namespace).
									Update(context.TODO(), &in, metav1.UpdateOptions{})
								if err != nil {
									log.WithField("ingress", result.Name).WithField("namesapce", result.Namespace).
										WithField("host", host).
										Error("Unable to update TLS section: ", err.Error())
								} else {
									// successful update of tls
									log.WithField("ingress", result.Name).
										WithField("namesapce", in.Namespace).
										WithField("host", host).
										WithField("tls", result.Spec.TLS).
										Info("Ingress TLS updated")
								}
							}
						}

					}
				}
			}
		}

		time.Sleep(10 * time.Second)
	}
}

func includesCertManagerAnnotation(in v1.Ingress, issuer ClusterIssuer) bool {
	// cert-manager.io/cluster-issuer: dev-cloud-issuer
	for a := range in.Annotations {
		if a == "cert-manager.io/cluster-issuer" && in.Annotations[a] == issuer.name {
			return true
		}
	}
	return false;
}

func includesHost(tlsList []v1.IngressTLS, host string) bool {
	for _, tls := range tlsList {
		for _, h := range tls.Hosts {
			if h == host {
				return true
			}
		}
	}
	return false
}
func getClusterIssuers() []ClusterIssuer {
	clusterIssuerNames := LookupEnv("CLUSTER_ISSUERS")

	var clusterIssuers []ClusterIssuer // an empty list
	for index, i := range strings.Split(clusterIssuerNames, ",") {
		envkey := "CLUSTER_ISSUER_" + strconv.Itoa(index+1) + "_REGEX"
		regex := LookupEnv(envkey)
		if regex == "" {
			panic("Env Variable " + envkey + " must exist")
		}
		clusterIssuers = append(clusterIssuers, ClusterIssuer{name: i, regex: regex})
	}
	return clusterIssuers
}

// LookupEnvOrString lookup ENV string with given key,
func LookupEnv(key string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	panic("Environment variable " + key + " must be provided!")
}
