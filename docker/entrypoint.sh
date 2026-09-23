#!/usr/bin/env bash

set -o errtrace -o nounset -o pipefail -o errexit

echo "uname -a: $(uname -a)"

PACKER=/bin/packer

setup_qemu() {
    # See also:
    #   * https://github.com/qemu/qemu/blob/master/scripts/qemu-binfmt-conf.sh
    #   * https://github.com/tonistiigi/binfmt/blob/master/cmd/binfmt/main.go
    #   * https://docs.kernel.org/admin-guide/binfmt-misc.html

    # mount binfmt_misc to be able to register qemu binaries
    mount binfmt_misc -t binfmt_misc /proc/sys/fs/binfmt_misc

    # reset
    find /proc/sys/fs/binfmt_misc -type f -name 'qemu-*' -exec sh -c 'echo -1 > "$1"' shell {} \;

    uname_m="$(uname -m)"
    if [ "$uname_m" == "aarch64" ]; then
        echo "Register qemu-x86_64"
        # shellcheck disable=SC2028
        echo ":qemu-x86_64:M::\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x3e\x00:\xff\xff\xff\xff\xff\xfe\xfe\x00\xff\xff\xff\xff\xff\xff\xff\xff\xfe\xff\xff\xff:/usr/bin/qemu-x86_64-static:F" > /proc/sys/fs/binfmt_misc/register
    else
        echo "Register qemu-aarch64"
        # shellcheck disable=SC2028
        echo ":qemu-aarch64:M::\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\xb7\x00:\xff\xff\xff\xff\xff\xff\xff\x00\xff\xff\xff\xff\xff\xff\xff\xff\xfe\xff\xff\xff:/usr/bin/qemu-aarch64-static:F" > /proc/sys/fs/binfmt_misc/register
    fi
    echo "Register qemu-arm"
    # shellcheck disable=SC2028
    echo ":qemu-arm:M::\x7fELF\x01\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x28\x00:\xff\xff\xff\xff\xff\xff\xff\x00\xff\xff\xff\xff\xff\xff\xff\xff\xfe\xff\xff\xff:/usr/bin/qemu-arm-static:F" > /proc/sys/fs/binfmt_misc/register
}

do_qemu_setup=${SETUP_QEMU:-true}
if [ "$do_qemu_setup" = true ]; then
    setup_qemu
fi

declare -a EXTRA_SYSTEM_PACKAGES=()
for arg do
    shift
    if [[ "${arg}" == -extra-system-packages=* ]]; then
        IFS=',' read -r -a EXTRA_SYSTEM_PACKAGES <<< "${arg//-extra-system-packages=}"
        continue
    fi
    set -- "$@" "${arg}"
done

if [ "${#EXTRA_SYSTEM_PACKAGES[@]}" -gt 0 ]; then
    echo "Installing extra system packages: ${EXTRA_SYSTEM_PACKAGES[*]}"
    apt-get update
    apt-get install -y --no-install-recommends "${EXTRA_SYSTEM_PACKAGES[@]}"
fi

for entry in ./packer-*; do
    if [[ -x "$entry" ]]; then
        echo "Please remove $entry and try again."
        exit 1
    fi
done

export DONT_SETUP_QEMU=1

# ensure packer plugin/cache directories exist
mkdir -p "${PACKER_PLUGIN_PATH}"
mkdir -p "${PACKER_CACHE_DIR}"

# register the bundled arm plugin (no-op if already installed at this version)
"${PACKER}" plugins install --path /bin/packer-plugin-builder-arm github.com/mkaczanowski/arm

echo running "${PACKER}" "${@}"

exec "${PACKER}" "${@}"
