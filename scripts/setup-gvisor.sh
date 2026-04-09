#!/bin/bash
# Instala gVisor (runsc) y lo registra con el Docker daemon
# Requiere: Linux x86_64, Docker instalado, permisos de sudo
set -e

ARCH=$(uname -m)
if [ "$ARCH" != "x86_64" ]; then
  echo "gVisor solo soporta x86_64. Arquitectura detectada: $ARCH"
  exit 1
fi

echo "Descargando gVisor..."
curl -fsSL https://storage.googleapis.com/gvisor/releases/release/latest/x86_64/runsc -o /tmp/runsc
curl -fsSL https://storage.googleapis.com/gvisor/releases/release/latest/x86_64/runsc.sha512 -o /tmp/runsc.sha512
sha512sum -c /tmp/runsc.sha512

sudo install -m 755 /tmp/runsc /usr/local/bin/runsc

echo "Registrando runsc con Docker..."
sudo tee /etc/docker/daemon.json > /dev/null <<'EOF'
{
  "runtimes": {
    "runsc": {
      "path": "/usr/local/bin/runsc"
    }
  }
}
EOF

sudo systemctl restart docker
echo "gVisor instalado y registrado. Verificando..."
docker run --runtime=runsc --rm hello-world && echo "OK: gVisor funciona correctamente"
