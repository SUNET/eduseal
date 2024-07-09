#! /bin/sh

set -e

service_names="apigw etcd1 etcd2 etcd3 etcd4 etcd5 sealer_1 sealer_2 validator_1 validator_2"

pki_dir="pki"

# Generate CA key and cert
cat > ${pki_dir}/rootCA.conf <<EOF
[req]
default_bits       = 2048
prompt             = no
default_md         = sha256
distinguished_name = dn

[dn]
C  = SE
O  = SUNET
OU = eduSeal_dev_rootCA
CN = eduSeal_dev_ca
EOF

if [ ! -f ./${pki_dir}/rootCA.key ]; then
    echo Creating Root CA

    openssl genrsa -out ${pki_dir}/rootCA.key 2048
    openssl req -x509 -new -nodes -key ${pki_dir}/rootCA.key -sha256 -days 3650 -out ${pki_dir}/rootCA.crt -config ${pki_dir}/rootCA.conf
fi

# Create leaf certificates for each service
create_leaf_cert() {
    service_name=${1}

    if [ ! -f ./${pki_dir}/${service_name}.key ]; then
    echo Creating leaf certificate for ${service_name}

    # Generate config files for openssl
    if [ ! -f ${pki_dir}/${service_name}.conf ]; then
	cat > ${pki_dir}/${service_name}.conf <<EOF
[req]
default_bits       = 2048
prompt             = no
default_md         = sha256
distinguished_name = dn

[dn]
C  = SE
O  = SUNET
OU = eduSeal_dev_rootCA
CN = ${service_name}.eduseal.docker
EOF
	conf_generated=1
    fi

    if [ ! -f ${pki_dir}/${service_name}.ext ]; then

	cat > ${pki_dir}/${service_name}.ext <<EOF
# v3.ext
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${service_name}.eduseal.docker
DNS.2 = ${service_name}
EOF
    if [ ${service_name} = "sealer_1" ] || [ ${service_name} = "sealer_2" ]; then
        grpc_dns="sealer.eduseal.docker"
        cat >> ${pki_dir}/${service_name}.ext <<EOF
DNS.3 = sealer.eduseal.docker
EOF
    fi

    if [ ${service_name} = "validator_1" ] || [ ${service_name} = "validator_2" ]; then
        grpc_dns="validator.eduseal.docker"
        cat >> ${pki_dir}/${service_name}.ext <<EOF
DNS.3 = validator.eduseal.docker
EOF
    fi
	ext_generated=1
    fi

    openssl req -new -sha256 -nodes -out ${pki_dir}/${service_name}.csr -newkey rsa:2048 -keyout ${pki_dir}/${service_name}.key -config ${pki_dir}/${service_name}.conf
    openssl x509 -req -in ${pki_dir}/${service_name}.csr -CA ${pki_dir}/rootCA.crt -CAkey ${pki_dir}/rootCA.key -CAcreateserial -out ${pki_dir}/${service_name}.crt -days 730 -sha256 -extfile ${pki_dir}/${service_name}.ext
    cat ${pki_dir}/${service_name}.key ${pki_dir}/${service_name}.crt ${pki_dir}/rootCA.crt > ${pki_dir}/${service_name}.pem

    # remove any generated config files
    if [ $conf_generated -eq 1 ]; then
	rm ${pki_dir}/${service_name}.conf
    fi
    if [ $ext_generated -eq 1 ]; then
	rm ${pki_dir}/${service_name}.ext
    fi
fi
}


for service_name in ${service_names}; do
        create_leaf_cert ${service_name}
done

create_sealing_cert() {
    if [ ! -f ./${pki_dir}/document_sealing.key ]; then
        echo Creating leaf certificate for document sealing

    # Generate config files for openssl
    if [ ! -f ${pki_dir}/${service_name}.conf ]; then
	    cat > ${pki_dir}/document_sealing.conf <<EOF
[req]
default_bits       = 2048
prompt             = no
default_md         = sha256
distinguished_name = dn

[dn]
C  = SE
O  = SUNET
OU = eduSeal_dev_rootCA
CN = Document Signing CA
EOF
	conf_generated=1
    fi

    if [ ! -f ${pki_dir}/document_sealing.ext ]; then
	    cat > ${pki_dir}/document_sealing.ext <<EOF
# v3.ext
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
EOF
	    ext_generated=1
    fi

    openssl req -new -sha256 -nodes -out ${pki_dir}/document_sealing.csr -newkey rsa:2048 -keyout ${pki_dir}/document_sealing.key -config ${pki_dir}/document_sealing.conf
    openssl x509 -req -in ${pki_dir}/document_sealing.csr -CA ${pki_dir}/rootCA.crt -CAkey ${pki_dir}/rootCA.key -CAcreateserial -out ${pki_dir}/document_sealing.crt -days 730 -sha256 -extfile ${pki_dir}/document_sealing.ext
    cat ${pki_dir}/document_sealing.key ${pki_dir}/document_sealing.crt ${pki_dir}/rootCA.crt > ${pki_dir}/document_sealing.pem

    printf "Converting keyfiles to DER format\n"
    openssl rsa -in  ${pki_dir}/document_sealing.key -outform DER -out ${pki_dir}/document_sealing_private.der

    # remove any generated config files
    if [ $conf_generated -eq 1 ]; then
	rm ${pki_dir}/document_sealing.conf
    fi
    if [ $ext_generated -eq 1 ]; then
	rm ${pki_dir}/document_sealing.ext
    fi
fi
}

create_sealing_cert