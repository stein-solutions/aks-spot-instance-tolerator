CERTDIR="./certs"

openssl genrsa -out ${CERTDIR}/ca.key 2048
openssl req -x509 -new -nodes -key ${CERTDIR}/ca.key -subj "/CN=admission_ca" -days 10000 -out ${CERTDIR}/ca.crt
openssl genrsa -out ${CERTDIR}/tls.key 2048
openssl req -new -key ${CERTDIR}/tls.key -subj "/CN=0.0.0.0" -addext "subjectAltName = IP:0.0.0.0" -out ${CERTDIR}/server.csr
echo "subjectAltName=IP:0.0.0.0" > altsubj.ext
openssl x509 -req -in ${CERTDIR}/server.csr -extfile altsubj.ext -CA ${CERTDIR}/ca.crt -CAkey ${CERTDIR}/ca.key -CAcreateserial -out ${CERTDIR}/tls.cert -days 10000
