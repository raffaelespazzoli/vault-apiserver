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

export VAULT_ADDR=http://$(oc get route vault -n vault -o jsonpath='{.spec.host}')
export VAULT_TOKEN=$(oc get secret vault-init -n vault -o jsonpath='{.data.root_token}'| base64 -d )
export HA_UNSEAL_KEY=$(oc get secret vault-init -n vault -o jsonpath='{.data.unseal_key}' | base64 -d)
vault operator unseal ${HA_UNSEAL_KEY}


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
# oc get configmap -n openshift-etcd etcd-ca-bundle -o jsonpath='{.data.ca-bundle\.crt}' > /tmp/ca-bundle.crt
# oc create configmap etcd-serving-ca -n vault-apiserver-dev --from-file /tmp/ca-bundle.crt
# oc get secret -n openshift-etcd etcd-client -o jsonpath='{.data.tls\.key}' | base64 -d > /tmp/tls.key
# oc get secret -n openshift-etcd etcd-client -o jsonpath='{.data.tls\.crt}' | base64 -d > /tmp/tls.crt
# oc create secret tls etcd-client -n vault-apiserver-dev --cert /tmp/tls.crt --key /tmp/tls.key
make install
kustomize build ./config/local-development | oc apply -f - -n vault-apiserver-dev
tilt up
```

test

```shell
oc create -f ./test/secretengine.yaml -n vault-apiserver-dev

```

secretengine
policy
policyBinding