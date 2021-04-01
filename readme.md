## Local Development

deploy and configure vault

```shell
helm upgrade vault hashicorp/vault -i --create-namespace -n vault -f ./deployments/vault-values.yaml
oc expose service vault --port=8200 -n vault
export VAULT_ADDR=http://$(oc get route vault -n vault -o jsonpath='{.spec.host}')
export HA_INIT_RESPONSE=$(vault operator init -format=json -key-shares 1 -key-threshold 1 -recovery-shares 1 -recovery-threshold 1)

HA_UNSEAL_KEY=$(echo "$HA_INIT_RESPONSE" | jq -r .unseal_keys_b64[0])
HA_VAULT_TOKEN=$(echo "$HA_INIT_RESPONSE" | jq -r .root_token)

echo "$HA_UNSEAL_KEY"
echo "$HA_VAULT_TOKEN"

#here we are saving these variable in a secret, this is probably not what you should do in a production environment
oc delete secret vault-init -n vault
oc create secret generic vault-init -n vault --from-literal=unseal_key=${HA_UNSEAL_KEY} --from-literal=root_token=${HA_VAULT_TOKEN}

vault operator unseal ${HA_UNSEAL_KEY}

export VAULT_TOKEN=$(oc get secret vault-init -n vault -o jsonpath='{.data.root_token}'| base64 -d )
vault auth enable -tls-skip-verify kubernetes 
export sa_secret_name=$(oc get sa vault -n vault -o jsonpath='{.secrets[*].name}' | grep -o '\b\w*\-token-\w*\b')
export api_url=https://kubernetes.default.svc:443
oc get secret ${sa_secret_name} -n vault -o jsonpath='{.data.ca\.crt}' | base64 -d > /tmp/ca.crt
vault write -tls-skip-verify auth/kubernetes/config token_reviewer_jwt="$(oc serviceaccounts get-token vault -n vault)" kubernetes_host=${api_url} kubernetes_ca_cert=@/tmp/ca.crt
vault write -tls-skip-verify auth/kubernetes/role/vault-apiserver-dev bound_service_account_names=default bound_service_account_namespaces=vault-apiserver-dev policies=vault-api-server
vault policy write -tls-skip-verify vault-api-server ./deployments/vault-apiserver-policy.hcl
```

deploy and develop with api-server

```shell
oc new-project vault-apiserver-dev
oc annotate namespace vault-apiserver-dev openshift.io/node-selector="node-role.kubernetes.io/master=" --overwrite
oc annotate namespace vault-apiserver-dev scheduler.alpha.kubernetes.io/defaultTolerations='[{"operator": "Exists", "effect": "NoSchedule", "key": "node-role.kubernetes.io/master"}]' --overwrite
oc get configmap -n openshift-etcd etcd-ca-bundle -o jsonpath='{.data.ca-bundle\.crt}' > /tmp/ca-bundle.crt
oc create configmap etcd-serving-ca -n vault-apiserver-dev --from-file /tmp/ca-bundle.crt
oc get secret -n openshift-etcd etcd-client -o jsonpath='{.data.tls\.key}' | base64 -d > /tmp/tls.key
oc get secret -n openshift-etcd etcd-client -o jsonpath='{.data.tls\.crt}' | base64 -d > /tmp/tls.crt
oc create secret tls etcd-client -n vault-apiserver-dev --cert /tmp/tls.crt --key /tmp/tls.key
make install
kustomize build ./config/local-development | oc apply -f - -n vault-apiserver-dev
tilt up
```

test

```shell
oc create -f ./test/secretengine.yaml -n vault-apiserver-dev

```

--audit-policy-file string                                Path to the file that defines the audit policy configuration.
      --audit-webhook-batch-buffer-size int                     The size of the buffer to store events before batching and writing. Only used in batch mode. (default 10000)
      --audit-webhook-batch-max-size int                        The maximum size of a batch. Only used in batch mode. (default 400)
      --audit-webhook-batch-max-wait duration                   The amount of time to wait before force writing the batch that hadn't reached the max size. Only used in batch mode. (default 30s)
      --audit-webhook-batch-throttle-burst int                  Maximum number of requests sent at the same moment if ThrottleQPS was not utilized before. Only used in batch mode. (default 15)
      --audit-webhook-batch-throttle-enable                     Whether batching throttling is enabled. Only used in batch mode. (default true)
      --audit-webhook-batch-throttle-qps float32                Maximum average number of batches per second. Only used in batch mode. (default 10)
      --audit-webhook-config-file string                        Path to a kubeconfig formatted file that defines the audit webhook configuration.
      --audit-webhook-initial-backoff duration                  The amount of time to wait before retrying the first failed request. (default 10s)
      --audit-webhook-mode string                               Strategy for sending audit events. Blocking indicates sending events should block server responses. Batch causes the backend to buffer and write events asynchronously. Known modes are batch,blocking,blocking-strict. (default "batch")
      --audit-webhook-truncate-enabled                          Whether event and batch truncating is enabled.
      --audit-webhook-truncate-max-batch-size int               Maximum size of the batch sent to the underlying backend. Actual serialized size can be several hundreds of bytes greater. If a batch exceeds this limit, it is split into several batches of smaller size. (default 10485760)
      --audit-webhook-truncate-max-event-size int               Maximum size of the audit event sent to the underlying backend. If the size of an event is greater than this number, first request and response are removed, and if this doesn't reduce the size enough, event is discarded. (default 102400)
      --audit-webhook-version string                            API group and version used for serializing audit events written to webhook. (default "audit.k8s.io/v1")
      --authentication-kubeconfig string                        kubeconfig file pointing at the 'core' kubernetes server with enough rights to create tokenreviews.authentication.k8s.io.
      --authentication-skip-lookup                              If false, the authentication-kubeconfig will be used to lookup missing authentication configuration from the cluster.
      --authentication-token-webhook-cache-ttl duration         The duration to cache responses from the webhook token authenticator. (default 10s)
      --authentication-tolerate-lookup-failure                  If true, failures to look up missing authentication configuration from the cluster are not considered fatal. Note that this can result in authentication that treats all requests as anonymous.
      --authorization-always-allow-paths strings                A list of HTTP paths to skip during authorization, i.e. these are authorized without contacting the 'core' kubernetes server.
      --authorization-kubeconfig string                         kubeconfig file pointing at the 'core' kubernetes server with enough rights to create subjectaccessreviews.authorization.k8s.io.
      --authorization-webhook-cache-authorized-ttl duration     The duration to cache 'authorized' responses from the webhook authorizer. (default 10s)
      --authorization-webhook-cache-unauthorized-ttl duration   The duration to cache 'unauthorized' responses from the webhook authorizer. (default 10s)
      --bind-address ip                                         The IP address on which to listen for the --secure-port port. The associated interface(s) must be reachable by the rest of the cluster, and by CLI/web clients. If blank or an unspecified address (0.0.0.0 or ::), all interfaces will be used. (default 0.0.0.0)
      --cert-dir string                                         The directory where the TLS certs are located. If --tls-cert-file and --tls-private-key-file are provided, this flag will be ignored. (default "apiserver.local.config/certificates")
      --client-ca-file string                                   If set, any request presenting a client certificate signed by one of the authorities in the client-ca-file is authenticated with an identity corresponding to the CommonName of the client certificate.
      --contention-profiling                                    Enable lock contention profiling, if profiling is enabled
      --default-watch-cache-size int                            Default watch cache size. If zero, watch cache will be disabled for resources that do not have a default watch size set. (default 100)
      --delete-collection-workers int                           Number of workers spawned for DeleteCollection call. These are used to speed up namespace cleanup. (default 1)
      --disable-admission-plugins strings                       admission plugins that should be disabled although they are in the default enabled plugins list (NamespaceLifecycle, MutatingAdmissionWebhook, ValidatingAdmissionWebhook). Comma-delimited list of admission plugins: MutatingAdmissionWebhook, NamespaceLifecycle, ValidatingAdmissionWebhook. The order of plugins in this flag does not matter.
      --egress-selector-config-file string                      File with apiserver egress selector configuration.
      --enable-admission-plugins strings                        admission plugins that should be enabled in addition to default enabled ones (NamespaceLifecycle, MutatingAdmissionWebhook, ValidatingAdmissionWebhook). Comma-delimited list of admission plugins: MutatingAdmissionWebhook, NamespaceLifecycle, ValidatingAdmissionWebhook. The order of plugins in this flag does not matter.
      --enable-garbage-collector                                Enables the generic garbage collector. MUST be synced with the corresponding flag of the kube-controller-manager. (default true)
      --encryption-provider-config string                       The file containing configuration for encryption providers to be used for storing secrets in etcd
      --etcd-cafile string                                      SSL Certificate Authority file used to secure etcd communication.
      --etcd-certfile string                                    SSL certification file used to secure etcd communication.
      --etcd-compaction-interval duration                       The interval of compaction requests. If 0, the compaction request from apiserver is disabled. (default 5m0s)
      --etcd-count-metric-poll-period duration                  Frequency of polling etcd for number of resources per type. 0 disables the metric collection. (default 1m0s)
      --etcd-db-metric-poll-interval duration                   The interval of requests to poll etcd and update metric. 0 disables the metric collection (default 30s)
      --etcd-healthcheck-timeout duration                       The timeout to use when checking etcd health. (default 2s)
      --etcd-keyfile string                                     SSL key file used to secure etcd communication.
      --etcd-prefix string                                      The prefix to prepend to all resource paths in etcd. (default "/registry/sample-apiserver")
      --etcd-servers strings                                    List of etcd servers to connect with (scheme://ip:port), comma separated.
      --etcd-servers-overrides strings                          Per-resource etcd servers overrides, comma separated. The individual override format: group/resource#servers, where servers are URLs, semicolon separated.
      --feature-gates mapStringBool                             A set of key=value pairs that describe feature gates for alpha/experimental features. Options are:
                                                                APIListChunking=true|false (BETA - default=true)
                                                                APIPriorityAndFairness=true|false (BETA - default=true)
                                                                APIResponseCompression=true|false (BETA - default=true)
                                                                APIServerIdentity=true|false (ALPHA - default=false)
                                                                AllAlpha=true|false (ALPHA - default=false)
                                                                AllBeta=true|false (BETA - default=false)
                                                                EfficientWatchResumption=true|false (ALPHA - default=false)
                                                                RemainingItemCount=true|false (BETA - default=true)
                                                                RemoveSelfLink=true|false (BETA - default=true)
                                                                ServerSideApply=true|false (BETA - default=true)
                                                                StorageVersionAPI=true|false (ALPHA - default=false)
                                                                StorageVersionHash=true|false (BETA - default=true)
                                                                ValidateProxyRedirects=true|false (BETA - default=true)
                                                                WarningHeaders=true|false (BETA - default=true)
  -h, --help                                                    help for this command
      --http2-max-streams-per-connection int                    The limit that the server gives to clients for the maximum number of streams in an HTTP/2 connection. Zero means to use golangs default. (default 1000)
      --kubeconfig string                                       kubeconfig file pointing at the 'core' kubernetes server.
      --log-flush-frequency duration                            Maximum number of seconds between log flushes (default 5s)
      --log_backtrace_at traceLocation                          when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                                          If non-empty, write log files in this directory
      --log_file string                                         If non-empty, use this log file
      --log_file_max_size uint                                  Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                                             log to standard error instead of files (default true)
      --master --kubeconfig                                     (Deprecated: switch to --kubeconfig) The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.
      --one_output                                              If true, only write logs to their native severity level (vs also writing to each lower severity level
      --permit-port-sharing                                     If true, SO_REUSEPORT will be used when binding the port, which allows more than one instance to bind on the same address and port. [default=false]
      --profiling                                               Enable profiling via web interface host:port/debug/pprof/ (default true)
      --requestheader-allowed-names strings                     List of client certificate common names to allow to provide usernames in headers specified by --requestheader-username-headers. If empty, any client certificate validated by the authorities in --requestheader-client-ca-file is allowed.
      --requestheader-client-ca-file string                     Root certificate bundle to use to verify client certificates on incoming requests before trusting usernames in headers specified by --requestheader-username-headers. WARNING: generally do not depend on authorization being already done for incoming requests.
      --requestheader-extra-headers-prefix strings              List of request header prefixes to inspect. X-Remote-Extra- is suggested. (default [x-remote-extra-])
      --requestheader-group-headers strings                     List of request headers to inspect for groups. X-Remote-Group is suggested. (default [x-remote-group])
      --requestheader-username-headers strings                  List of request headers to inspect for usernames. X-Remote-User is common. (default [x-remote-user])
      --secure-port int                                         The port on which to serve HTTPS with authentication and authorization. If 0, dont serve HTTPS at all. (default 443)
      --skip_headers                                            If true, avoid header prefixes in the log messages
      --skip_log_headers                                        If true, avoid headers when opening log files
      --standalone-debug-mode                                   Under the local-debug mode the apiserver will allow all access to its resources without authorizing the requests, this flag is only intended for debugging in your workstation and the apiserver will be crashing if its binding address is not 127.0.0.1.
      --stderrthreshold severity                                logs at or above this threshold go to stderr (default 2)
      --storage-backend string                                  The storage backend for persistence. Options: 'etcd3' (default).
      --storage-media-type string                               The media type to use to store objects in storage. Some resources or storage backends may only support a specific media type and will ignore this setting. (default "application/json")
      --tls-cert-file string                                    File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated after server cert). If HTTPS serving is enabled, and --tls-cert-file and --tls-private-key-file are not provided, a self-signed certificate and key are generated for the public address and saved to the directory specified by --cert-dir.
      --tls-cipher-suites strings                               Comma-separated list of cipher suites for the server. If omitted, the default Go cipher suites will be used.
                                                                Preferred values: TLS_AES_128_GCM_SHA256, TLS_AES_256_GCM_SHA384, TLS_CHACHA20_POLY1305_SHA256, TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA, TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256, TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA, TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA, TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA, TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305, TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256, TLS_RSA_WITH_3DES_EDE_CBC_SHA, TLS_RSA_WITH_AES_128_CBC_SHA, TLS_RSA_WITH_AES_128_GCM_SHA256, TLS_RSA_WITH_AES_256_CBC_SHA, TLS_RSA_WITH_AES_256_GCM_SHA384.
                                                                Insecure values: TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, TLS_ECDHE_ECDSA_WITH_RC4_128_SHA, TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256, TLS_ECDHE_RSA_WITH_RC4_128_SHA, TLS_RSA_WITH_AES_128_CBC_SHA256, TLS_RSA_WITH_RC4_128_SHA.
      --tls-min-version string                                  Minimum TLS version supported. Possible values: VersionTLS10, VersionTLS11, VersionTLS12, VersionTLS13
      --tls-private-key-file string                             File containing the default x509 private key matching --tls-cert-file.
      --tls-sni-cert-key namedCertKey                           A pair of x509 certificate and private key file paths, optionally suffixed with a list of domain patterns which are fully qualified domain names, possibly with prefixed wildcard segments. The domain patterns also allow IP addresses, but IPs should only be used if the apiserver has visibility to the IP address requested by a client. If no domain patterns are provided, the names of the certificate are extracted. Non-wildcard matches trump over wildcard matches, explicit domain patterns trump over extracted names. For multiple key/certificate pairs, use the --tls-sni-cert-key multiple times. Examples: "example.crt,example.key" or "foo.crt,foo.key:*.foo.com,foo.com". (default [])
  -v, --v Level                                                 number for the log level verbosity
      --vmodule moduleSpec                                      comma-separated list of pattern=N settings for file-filtered logging
      --watch-cache                                             Enable watch caching in the apiserver (default true)
      --watch-cache-sizes strings                               Watch cache size settings for some resources (pods, nodes, etc.), comma separated. The individual setting format: resource[.group]#size, where resource is lowercase plural (no version), group is omitted for resources of apiVersion v1 (the legacy core API) and included for others, and size is a number. It takes effect when watch-cache is enabled. Some resources (replicationcontrollers, endpoints, nodes, pods, services, apiservices.apiregistration.k8s.io) have system defaults set by heuristics, others default to default-watch-cache-size
