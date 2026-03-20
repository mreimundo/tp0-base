import socket
import logging
import signal
from common.utils import Bet, store_bets
from common.protocol import recv_batch, send_batch_ack
from common.utils import Bet, store_bets

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._running = True
        signal.signal(signal.SIGTERM, self.__handle_sigterm)

    def __handle_sigterm(self, sig, frame):
        logging.info("action: receive_sigterm | result: success")
        self._running = False
        self._server_socket.close()
        logging.info("action: close_server_socket | result: success")

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        while self._running:
            try:
                client_sock = self.__accept_new_connection()
                self.__handle_client_connection(client_sock)
            except OSError:
                # accept() lanza OSError cuando el socket se cierra por SIGTERM
                break
        logging.info("action: server_shutdown | result: success")

    def __handle_client_connection(self, client_sock):
        try:
            while True:
                bets_data = recv_batch(client_sock)
                if bets_data is None:
                    break  # cliente terminó de enviar

                bets = [Bet(b['agency'], b['first_name'], b['last_name'],
                            b['document'], b['birthdate'], b['number'])
                        for b in bets_data]
                store_bets(bets)
                logging.info(f'action: apuesta_recibida | result: success | cantidad: {len(bets)}')
                send_batch_ack(client_sock, True)
        except OSError as e:
            logging.error(f'action: apuesta_recibida | result: fail | error: {e}')
            send_batch_ack(client_sock, False)
        finally:
            client_sock.close()
            logging.info('action: close_client_socket | result: success')

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
    
    def send_all(sock, data: bytes):
        """
        Send all data to the socket
        
        short-write is possible when the socket buffer is full,
        so we need to loop until all data is sent
        """
        total = 0
        while total < len(data):
            sent = sock.send(data[total:])
            if sent == 0:
                raise OSError("Connection broken")
            total += sent

    def recv_all(sock, n: int) -> bytes:
        """
        Read n bytes from the socket
        
        Avoids short-read by looping until all n bytes are read
        """
        data = b''
        while len(data) < n:
            chunk = sock.recv(n - len(data))
            if not chunk:
                raise OSError("Connection broken")
            data += chunk
        return data
