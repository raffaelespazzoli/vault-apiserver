# add full access to the /sys/mounts for secret engine configurations
path "/sys/mounts/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "/sys/mounts" {
  capabilities = ["read", "list"]
}