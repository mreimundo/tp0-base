#!/bin/bash


# args esperados (chequeo entrada)
if [ "$#" -ne 2 ]; then
  echo "Uso: $0 <archivo_salida> <cantidad_clientes>"
  exit 1
fi

file_out=$1
clients_qty=$2

# creamos el header para el compose y se lo insertamos al archivo de salida
cat > "$file_out" <<EOL
name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
      - LOGGING_LEVEL=DEBUG
    networks:
      - testing_net
EOL

# le sumo al archivo de salida la config por client
i=1
while [ $i -le $clients_qty ]; do
  cat >> "$file_out" <<EOL

  client$i:
    container_name: client$i
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=$i
      - CLI_LOG_LEVEL=DEBUG
    networks:
      - testing_net
    depends_on:
      - server
EOL
  i=$((i + 1))
done

# le agrego al archivo de salida la parte de red
cat >> "$file_out" <<EOL

networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
EOL

echo "Archivo Docker Compose generado en: $file_out"