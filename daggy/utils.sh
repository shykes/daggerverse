export DAGGER_MODULE=github.com/shykes/core

decode() {
    local input="$1"
    # Strip the leading URL scheme if present
    local stripped_scheme="${input#*://}"
    # Replace a trailing "+" with "=" if present, then replace "++" with "=="
    local processed="${stripped_scheme/%+/=}"
    echo "${processed/%++/==}"
}

encode() {

    read -r input
    # local input="$1"
    # Replace a trailing "=" with "+" if present, then replace "==" with "++"
    local processed="${input/%=/+}"
    processed="${processed/%==/++}"
    # Prepend the "data://" URL scheme
    echo "data://${processed}"
}
