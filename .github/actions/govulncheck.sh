#!/usr/bin/env bash

set -eo pipefail

main(){
    local go_package="$1"
    govulncheck -json "$go_package" > vulns.json

    jq -r '.finding | select( (.osv != null) and (.trace[0].function != null) ) | .osv ' < vulns.json > vulns_osv_ids.txt

    ignore GO-2026-4340 "BRIDGE-466 TLS 1.3 specific, network-local attacker may disclose minor information before encryption level changes"
    ignore GO-2026-4337 "BRIDGE-466 crypto/tls if underlying Config has ClientCAs or RootCAs fields mutated between handshake and resumed handshake  the resumed one may succeed when it should have failed."
    ignore GO-2026-4440 "BRIDGE-466 html.Parse has quadratic parsing complexity with certain inputs can lead to DoS"
    ignore GO-2026-4441 "BRIDGE-466 html.Parse has infinite parsing loop with certain inputs can lead to DoS"

    has_vulns

    echo
    echo "No new vulnerabilities found."
}

ignore(){
    echo "ignoring $1 fix: $2"
    cp vulns_osv_ids.txt tmp
    grep -v "$1" < tmp > vulns_osv_ids.txt || true
    rm tmp
}

has_vulns(){
    has=false
    while read -r osv; do
        jq \
            --arg osvid "$osv" \
            '.osv | select ( .id == $osvid) | {"id":.id, "ranges": .affected[0].ranges,  "import": .affected[0].ecosystem_specific.imports[0].path}' \
            < vulns.json
        has=true
    done < vulns_osv_ids.txt

    if [ "$has" == true ]; then
        echo
        echo "Vulnerability found"
        return 1
    fi
}

main
