#!/bin/bash


# args esperados (chequeo entrada)
if [ "$#" -ne 2 ]; then
  echo "Uso: $0 <archivo_salida> <cantidad_clientes>"
  exit 1
fi

file_out="$1"
clients_qty=$2

# creamos el header para el compose y se lo insertamos al archivo de salida
# updates ej2:
#   - le agrego el volumen para el config.ini del server, entonces se puede modificar sin rebuildear la imagen
#   - quito la env var de log_level debug porque el test pide info
cat > "$file_out" <<EOF
name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      PYTHONUNBUFFERED: 1
    networks:
      - testing_net
    volumes:
      - ./server/config.ini:/config.ini
EOF

# le sumo al archivo de salida la config por client
# update ej2:
#   - le agrego el volumen para el config.yaml de cada client, entonces se puede modificar sin rebuildear la imagen
#   - quito la env var de log_level debug porque el test pide info
for ((i=1;i<=clients_qty;i++)); do
cat >> "$file_out" <<EOF

  client$i:
    container_name: client$i
    image: client:latest
    entrypoint: /client
    environment:
      CLI_ID: $i
    volumes:
      - ./client/config.yaml:/config.yaml
      - ./.data/agency-$i.csv:/agency.csv
    networks:
      - testing_net
    depends_on:
      server:
        condition: service_started
EOF
done

# le agrego al archivo de salida la parte de red
cat >> "$file_out" <<EOF

networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
EOF

echo "Archivo Docker Compose generado en: $file_out"