#!/bin/bash
MESSAGE="ping" #ping porque es un echo server, podría ser cualquier cosa


# ejecutamos el comando para testear el echo server, le pasamos el mensaje por stdin (0) y lo recibimos por stdout (1)
# el nombre de la red lo conformamos con el name que le damos al compose y testing_net porque lo definimos así
RESPONSE=$(echo "$MESSAGE" | docker run --rm -i \
    --network tp0_testing_net \
    busybox nc server 12345)

if [ "$RESPONSE" = "$MESSAGE" ]; then
    echo "action: test_echo_server | result: success"
else
    echo "action: test_echo_server | result: fail"
fi