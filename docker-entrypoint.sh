#!/bin/sh
set -e

# Linux'ta bind mount edilen ./data klasörünü Docker root sahipliğinde oluşturur.
# Container root olarak başlarsa veri dizini uygulama kullanıcısına devredilir,
# ardından uygulama root yetkisi bırakılarak (su-exec) gowatch kullanıcısıyla çalıştırılır.
if [ "$(id -u)" = "0" ]; then
	mkdir -p /app/data
	find /app/data \! -user gowatch -exec chown gowatch:gowatch {} +
	exec su-exec gowatch "$@"
fi

exec "$@"
