#!/bin/bash

set -e

NAMESPACE=${NAMESPACE:-default}
SERVICE=${SERVICE:-aks-spot-instance-tolerator-webhook-svc}
SECRET=${SECRET:-aks-spot-instance-tolerator-webhook}
CERTDIR=${CERTDIR:-webhook-certs}

mkdir -p ${CERTDIR}

# openssl genrsa -out ${CERTDIR}/ca.key 2048
# openssl req -x509 -new -nodes -key ${CERTDIR}/ca.key -subj "/CN=admission_ca" -days 10000 -out ${CERTDIR}/ca.crt
# openssl genrsa -out ${CERTDIR}/server.key 2048
# openssl req -new -key ${CERTDIR}/server.key -subj "/CN=${SERVICE}.${NAMESPACE}.svc" -out ${CERTDIR}/server.csr
# openssl x509 -req -in ${CERTDIR}/server.csr -CA ${CERTDIR}/ca.crt -CAkey ${CERTDIR}/ca.key -CAcreateserial -out ${CERTDIR}/server.crt -days 10000

# kubectl create secret tls ${SECRET} \
#     --cert=${CERTDIR}/server.crt \
#     --key=${CERTDIR}/server.key \
#     --namespace=${NAMESPACE} \


# Generate the CA key and certificate
openssl genrsa -out ${CERTDIR}/ca.key 2048
openssl req -x509 -new -nodes -key ${CERTDIR}/ca.key -subj "/CN=admission_ca" -days 10000 -out ${CERTDIR}/ca.crt

# Generate the server key
openssl genrsa -out ${CERTDIR}/server.key 2048

# Generate the CSR with SANs
cat <<EOF > ${CERTDIR}/csr.conf
[ req ]
default_bits = 2048
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[ dn ]
CN = ${SERVICE}.${NAMESPACE}.svc

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = ${SERVICE}
DNS.2 = ${SERVICE}.${NAMESPACE}
DNS.3 = ${SERVICE}.${NAMESPACE}.svc
DNS.4 = ${SERVICE}.${NAMESPACE}.svc.cluster.local
EOF

openssl req -new -key ${CERTDIR}/server.key -out ${CERTDIR}/server.csr -config ${CERTDIR}/csr.conf

# Generate the server certificate
cat <<EOF > ${CERTDIR}/cert.conf
[ v3_ext ]
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = ${SERVICE}
DNS.2 = ${SERVICE}.${NAMESPACE}
DNS.3 = ${SERVICE}.${NAMESPACE}.svc
DNS.4 = ${SERVICE}.${NAMESPACE}.svc.cluster.local
EOF

openssl x509 -req -in ${CERTDIR}/server.csr -CA ${CERTDIR}/ca.crt -CAkey ${CERTDIR}/ca.key -CAcreateserial -out ${CERTDIR}/server.crt -days 10000 -extensions v3_ext -extfile ${CERTDIR}/cert.conf
