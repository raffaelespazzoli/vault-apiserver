# add full access to the /sys/mounts for secret engine configurations
path "/sys/mounts/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "/sys/mounts" {
  capabilities = ["read", "list"]
}

path "/sys/policy/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "/sys/policy" {
  capabilities = ["read", "list"]
}

path "/auth/kubernetes/role/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "/auth/kubernetes/role" {
  capabilities = ["read", "list"]
}