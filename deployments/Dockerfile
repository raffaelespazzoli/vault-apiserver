FROM registry.access.redhat.com/ubi8/ubi-minimal
WORKDIR /app
RUN mkdir -p /app/data && chmod 0777 /app/data
ADD build build
ENTRYPOINT build/vault-apiserver