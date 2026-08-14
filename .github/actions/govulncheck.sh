#!/usr/bin/env bash

set -eo pipefail

main(){
    local go_package="$1"
    govulncheck -json "$go_package" > vulns.json

    jq -r '.finding | select( (.osv != null) and (.trace[0].function != null) ) | .osv ' < vulns.json > vulns_osv_ids.txt

    ignore GO-2026-4550 "Indirect import from goopengpg. Waiting for fix on their side"
    ignore GO-2026-5026 "BRIDGE-622 The ToASCII and ToUnicode functions incorrectly accept Punycode-encoded labels that decode to an ASCII-only label."
    ignore GO-2026-5972 "BRIDGE-622 Enforce a recursion limit in Unmarshal to prevent stack exhaustion when parsing deeply-nested, recursive structures."
    ignore GO-2026-6088 "BRIDGE-622 Previously, DecodeElement would reset the depth counter causing it to never fire; this could lead to stack exhaustion."
    ignore GO-2026-6090 "BRIDGE-622 Handshake messages are always considered state-advancing, a malicious client can keep sending these messages to force the server to do key derivation operations."
    ignore GO-2026-6218 "BRIDGE-622 Path resolution operates on a byte buffer using index-based backtracking for '..' segments, eliminating the quadratic time complexity and significantly reducing memory allocations."

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
