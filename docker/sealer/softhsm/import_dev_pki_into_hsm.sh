#!/usr/bin/env bash

printf "import PKI\n"

set -e

pki_folder="/opt/sunet/pki"
if [[ ! -d  ${pki_folder} ]]; then
    mkdir -p ${pki_folder}
fi

# Read PKCS#11 settings from environment variables.
: "${pkcs11_pin:?ERROR: pkcs11_pin not set}"
: "${pkcs11_module:?ERROR: pkcs11_module not set}"
: "${pkcs11_label:?ERROR: pkcs11_label not set}"
: "${pkcs11_key_label:?ERROR: pkcs11_key_label not set}"
: "${pkcs11_cert_label:?ERROR: pkcs11_cert_label not set}"

printf "PKCS#11 config: module=%s label=%s\n" "${pkcs11_module}" "${pkcs11_label}"

clean_hsm() {
    printf "clean hsm\n"
    softhsm2-util --delete-token --token "${pkcs11_label}"
    rm  -r /var/lib/softhsm/tokens/*
    printf "done!\n"
}

init_hsm() {
    printf "Init hsm\n"
    pkcs11-tool --module "${pkcs11_module}" --init-token --init-pin --login --pin "${pkcs11_pin}" --so-pin "${pkcs11_pin}" --label "${pkcs11_label}"
    printf "done!\n"
}

import_private_key() {
    printf "import private key\n"
    pkcs11-tool --module "${pkcs11_module}" -l --pin "${pkcs11_pin}" --write-object "${pki_folder}"/sealing.der --type privkey --id 1001 --label "${pkcs11_key_label}"
    printf "done!\n"
}

import_cert() {
    printf "import certificate\n"
    pkcs11-tool --module "${pkcs11_module}" -l --pin "${pkcs11_pin}" --write-object "${pki_folder}"/sealing.crt --type cert --id 2002 --label "${pkcs11_cert_label}"
    printf "done!\n"
}

pkcs11_list() {
    printf "print pkcs11 objects\n"
    pkcs11-tool --module "${pkcs11_module}" -L --pin "${pkcs11_pin}" -T -O -I
    printf "done!\n"
}

show_slots() {
    softhsm2-util --show-slots
}

create_run_once() {
    touch "${pki_folder}"/.run_once_hsm
    date > "${pki_folder}"/.run_once_hsm
}

prevent_run_more_than_once() {
    if [[ -f  "${pki_folder}"/.run_once_hsm ]]; then
        printf "this script can only run once!\n"
        exit 0
    fi
}

#prevent_run_more_than_once
#clean_hsm
init_hsm

import_private_key
import_cert

pkcs11_list
show_slots
create_run_once