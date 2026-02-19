#!/usr/bin/env bash
set -euo pipefail

DATA_DIR="${DATA_DIR:-/data}"
EULA="${EULA:-true}"

# Choose one: fabric | forge | quilt | paper | vanilla
LOADER="${LOADER:-vanilla}"
MC_VERSION="${MC_VERSION:-1.21.1}"

# Memory
JAVA_XMS="${JAVA_XMS:-1G}"
JAVA_XMX="${JAVA_XMX:-2G}"

# RCON (Minecraft has built-in rcon)
ENABLE_RCON="${ENABLE_RCON:-true}"
RCON_PORT="${RCON_PORT:-25575}"
RCON_PASSWORD="${RCON_PASSWORD:-change-me}"

# Where to put the runnable jar
SERVER_JAR="${SERVER_JAR:-${DATA_DIR}/server.jar}"

# Optional bootstrap source (GitHub repo)
GIT_BOOTSTRAP_REPO="${GIT_BOOTSTRAP_REPO:-}"
GIT_BOOTSTRAP_REF="${GIT_BOOTSTRAP_REF:-main}"
GIT_BOOTSTRAP_PATH="${GIT_BOOTSTRAP_PATH:-}"
GIT_BOOTSTRAP_TOKEN="${GIT_BOOTSTRAP_TOKEN:-}"

mkdir -p "${DATA_DIR}"

bootstrap_from_git () {
  if [[ -z "${GIT_BOOTSTRAP_REPO}" ]]; then
    return
  fi

  local tmp_dir repo_url src_dir
  tmp_dir="$(mktemp -d)"
  repo_url="${GIT_BOOTSTRAP_REPO}"

  if [[ -n "${GIT_BOOTSTRAP_TOKEN}" && "${repo_url}" =~ ^https:// ]]; then
    repo_url="${repo_url/https:\/\//https:\/\/${GIT_BOOTSTRAP_TOKEN}@}"
  fi

  echo "Bootstrapping server files from git repo..."
  git clone --depth 1 --branch "${GIT_BOOTSTRAP_REF}" "${repo_url}" "${tmp_dir}/repo"

  src_dir="${tmp_dir}/repo"
  if [[ -n "${GIT_BOOTSTRAP_PATH}" ]]; then
    src_dir="${src_dir}/${GIT_BOOTSTRAP_PATH}"
  fi

  if [[ ! -d "${src_dir}" ]]; then
    echo "GIT_BOOTSTRAP_PATH not found in repo: ${GIT_BOOTSTRAP_PATH}"
    rm -rf "${tmp_dir}"
    exit 1
  fi

  shopt -s dotglob nullglob
  for item in "${src_dir}"/*; do
    if [[ "$(basename "${item}")" == ".git" ]]; then
      continue
    fi
    cp -a "${item}" "${DATA_DIR}/"
  done
  shopt -u dotglob nullglob

  rm -rf "${tmp_dir}"
}

# If server jar is missing, optionally pull initial files from git first.
if [[ ! -f "${SERVER_JAR}" ]]; then
  bootstrap_from_git
fi

# If a known jar exists, normalize it to SERVER_JAR before installer fallback.
if [[ ! -f "${SERVER_JAR}" ]]; then
  if [[ -f "${DATA_DIR}/fabric-server-launch.jar" ]]; then
    ln -sf "${DATA_DIR}/fabric-server-launch.jar" "${SERVER_JAR}"
  else
    FORGE_EXISTING_JAR="$(ls -1 "${DATA_DIR}"/forge-*-server.jar 2>/dev/null | head -n 1 || true)"
    if [[ -n "${FORGE_EXISTING_JAR}" ]]; then
      ln -sf "${FORGE_EXISTING_JAR}" "${SERVER_JAR}"
    fi
  fi
fi

# eula
if [[ "${EULA}" == "true" ]]; then
  echo "eula=true" > "${DATA_DIR}/eula.txt"
fi

# Ensure server.properties has RCON configured (only if enabled)
PROPS="${DATA_DIR}/server.properties"
if [[ ! -f "${PROPS}" ]]; then
  touch "${PROPS}"
fi

set_prop () {
  local key="$1" val="$2"
  if grep -qE "^${key}=" "${PROPS}"; then
    sed -i "s|^${key}=.*|${key}=${val}|" "${PROPS}"
  else
    echo "${key}=${val}" >> "${PROPS}"
  fi
}

if [[ "${ENABLE_RCON}" == "true" ]]; then
  set_prop "enable-rcon" "true"
  set_prop "rcon.port" "${RCON_PORT}"
  set_prop "rcon.password" "${RCON_PASSWORD}"
else
  set_prop "enable-rcon" "false"
fi

# Download/install server jar if missing
if [[ ! -f "${SERVER_JAR}" ]]; then
  echo "No server jar found at ${SERVER_JAR}. Installing ${LOADER} for MC ${MC_VERSION}..."

  case "${LOADER}" in
    vanilla)
      # Mojang vanilla server URL changes; better to pin and provide SERVER_URL.
      if [[ -z "${SERVER_URL:-}" ]]; then
        echo "Set SERVER_URL for vanilla, or use fabric/forge/quilt."
        exit 1
      fi
      curl -fsSL "${SERVER_URL}" -o "${SERVER_JAR}"
      ;;

    fabric)
      # Uses Fabric installer to generate server jar/launch config.
      FABRIC_INSTALLER_VERSION="${FABRIC_INSTALLER_VERSION:-1.0.1}"
      FABRIC_LOADER_VERSION="${FABRIC_LOADER_VERSION:-0.16.7}"
      INSTALLER_JAR="/tmp/fabric-installer.jar"
      curl -fsSL "https://maven.fabricmc.net/net/fabricmc/fabric-installer/${FABRIC_INSTALLER_VERSION}/fabric-installer-${FABRIC_INSTALLER_VERSION}.jar" \
        -o "${INSTALLER_JAR}"
      # Installs into DATA_DIR; creates fabric-server-launch.jar etc.
      java -jar "${INSTALLER_JAR}" server \
        -downloadMinecraft \
        -mcversion "${MC_VERSION}" \
        -loader "${FABRIC_LOADER_VERSION}" \
        -dir "${DATA_DIR}"
      # Fabric typically creates fabric-server-launch.jar
      if [[ -f "${DATA_DIR}/fabric-server-launch.jar" ]]; then
        ln -sf "${DATA_DIR}/fabric-server-launch.jar" "${SERVER_JAR}"
      fi
      ;;

    quilt)
      # Quilt installer can do similar; requires a SERVER_URL or installer flow.
      if [[ -z "${SERVER_URL:-}" ]]; then
        echo "For quilt, easiest is to provide SERVER_URL to your quilt server jar."
        exit 1
      fi
      curl -fsSL "${SERVER_URL}" -o "${SERVER_JAR}"
      ;;

    forge)
      # Forge usually needs the installer and a run script.
      # Best practice: provide FORGE_INSTALLER_URL for the exact version you want.
      if [[ -z "${FORGE_INSTALLER_URL:-}" ]]; then
        echo "Set FORGE_INSTALLER_URL to a specific Forge installer jar."
        exit 1
      fi
      INSTALLER_JAR="/tmp/forge-installer.jar"
      curl -fsSL "${FORGE_INSTALLER_URL}" -o "${INSTALLER_JAR}"
      # Install into DATA_DIR (creates libraries, run scripts)
      (cd "${DATA_DIR}" && java -jar "${INSTALLER_JAR}" --installServer)
      # Forge often outputs a runnable jar with "forge-*-server.jar"
      FORGE_JAR="$(ls -1 "${DATA_DIR}"/forge-*-server.jar 2>/dev/null | head -n 1 || true)"
      if [[ -n "${FORGE_JAR}" ]]; then
        ln -sf "${FORGE_JAR}" "${SERVER_JAR}"
      fi
      ;;

    *)
      echo "Unknown LOADER=${LOADER}"
      exit 1
      ;;
  esac
fi

# Ensure ownership
chown -R minecraft:minecraft "${DATA_DIR}"

echo "Starting Minecraft (${LOADER}) with ${JAVA_XMS}/${JAVA_XMX}..."
exec gosu minecraft java \
  -Xms"${JAVA_XMS}" -Xmx"${JAVA_XMX}" \
  -jar "${SERVER_JAR}" nogui
