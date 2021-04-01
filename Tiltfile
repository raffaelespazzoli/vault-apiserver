# -*- mode: Python -*-

# For more on Extensions, see: https://docs.tilt.dev/extensions.html
load('ext://restart_process', 'docker_build_with_restart')

compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/vault-apiserver ./'

local_resource(
  'vault-apiserver-compile',
  compile_cmd,
  deps=['./main.go','./api','./vaultstorage'])


custom_build(
  'quay.io/raffaelespazzoli/vault-apiserver',
  'buildah bud -t $EXPECTED_REF -f deployments/Dockerfile .  && buildah push $EXPECTED_REF $EXPECTED_REF',
  entrypoint=['/app/build/vault-apiserver'],
  deps=['./build'],
  live_update=[
    sync('./build', '/app/build'),
  ],
  skips_local_docker=True,
)

allow_k8s_contexts('vault-apiserver-dev/api-tmp-raffa-demo-red-chesterfield-com:6443/raffa')
k8s_yaml(['deployments/deployment.yaml','deployments/service.yaml','deployments/apiservice.yaml'])
k8s_resource('vault-apiserver',resource_deps=['vault-apiserver-compile'])