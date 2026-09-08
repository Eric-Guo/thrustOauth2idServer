#!/usr/bin/env bash
# Usage: ./yamlParser.sh <yaml_file> <key_path>
# Example: ./yamlParser.sh config.yaml '.http.tls.keyFile'

yaml_file="$1"
key_path="$2"

if [[ -z "$yaml_file" || -z "$key_path" ]]; then
  echo "Usage: $0 <yaml_file> <key_path>"
  exit 1
fi

if [[ ! -f "$yaml_file" ]]; then
  echo "File not found: $yaml_file"
  exit 1
fi

key_path="${key_path#.}"

# Read scalar configuration paths using POSIX awk (including macOS awk).
awk -v wanted="$key_path" '
/^[[:space:]]*#|^[[:space:]]*$/ { next }
{
    match($0, /^[[:space:]]*/)
    indent = RLENGTH
    line = substr($0, indent + 1)
    colon = index(line, ":")
    if (!colon) next
    key = substr(line, 1, colon - 1)
    sub(/[[:space:]]+$/, "", key)
    while (depth > 0 && indent <= levels[depth]) depth--
    depth++
    levels[depth] = indent
    names[depth] = key
    path = names[1]
    for (i = 2; i <= depth; i++) path = path "." names[i]
    if (path != wanted) next
    value = substr(line, colon + 1)
    sub(/^[[:space:]]+/, "", value)
    quote = substr(value, 1, 1)
    if (quote == "\"" || quote == sprintf("%c", 39)) {
        value = substr(value, 2)
        end = index(value, quote)
        if (end) value = substr(value, 1, end - 1)
    } else {
        sub(/[[:space:]]+#.*$/, "", value)
        sub(/[[:space:]]+$/, "", value)
    }
    print value
    exit
}
' "$yaml_file"
