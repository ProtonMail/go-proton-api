#!/usr/bin/env bash

set -eo pipefail

main(){
    local go_package="$1"
    govulncheck -json "$go_package" > vulns.json

    jq -r '.finding | select( (.osv != null) and (.trace[0].function != null) ) | .osv ' < vulns.json > vulns_osv_ids.txt

    ignore GO-2026-4550 "Indirect import from goopengpg. Waiting for fix on their side"
    ignore GO-2026-4918 "BRIDGE-554 net/http & /x/net/http2 infinite loop while processing HTTP/2 setting frames" 
    ignore GO-2026-4971 "BRIDGE-554 net Dial and LookupPort panics on Windows when receiving input with NUL (0)"
    ignore GO-2026-4980 "BRIDGE-554 html/template escape data passed to <script> block"
    ignore GO-2026-4982 "BRIDGE-554 html/template XSS vector if url contains ASCII whitespace"
    ignore GO-2026-4986 "BRIDGE-554 net/mail triggers CPU exhaustion and memory allocations"
    ignore GO-2026-5025 "BRIDGE-554 /x/net parsing HTML using Render can lead to unexpected HTML tree."
    ignore GO-2026-5026 "BRIDGE-554 /x/net ToASCII and ToUnicode incorrectly accept punycode-encoded labels."
    ignore GO-2026-5027 "BRIDGE-554 /x/net parsing HTML using Render can lead to unexpected HTML tree."
    ignore GO-2026-5028 "BRIDGE-554 /x/net parsing arbitrary HTML  can consume excessive CPU time"
    ignore GO-2026-5029 "BRIDGE-554 /x/net parsing HTML using Render can lead to unexpected HTML tree."
    ignore GO-2026-5030 "BRIDGE-554 /x/net parsing HTML using Render can lead to unexpected HTML tree."

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
