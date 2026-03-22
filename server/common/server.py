import socket
import logging
import signal
from common.utils import Bet, store_bets, load_bets, has_won
from common.protocol import (
    recv_msg_type, recv_batch_payload, recv_done, recv_query,
    send_batch_ack, send_done_ack, send_winners, send_not_ready,
    MSG_BATCH, MSG_DONE, MSG_QUERY
)
TOTAL_AGENCIES = 5

class Server:
    def __init__(self, port, listen_backlog):
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._running = True
        self._agencies_done = set()
        self._lottery_done  = False
        self._winners = {}
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
                msg_type = recv_msg_type(client_sock)

                if msg_type == MSG_BATCH:
                    bets_data = recv_batch_payload(client_sock)
                    bets = [Bet(b['agency'], b['first_name'], b['last_name'],
                                b['document'], b['birthdate'], b['number'])
                            for b in bets_data]
                    store_bets(bets)
                    logging.info(f'action: apuesta_recibida | result: success | cantidad: {len(bets)}')
                    send_batch_ack(client_sock, True)

                elif msg_type == MSG_DONE:
                    agency_id = recv_done(client_sock)
                    self._agencies_done.add(agency_id)
                    if len(self._agencies_done) == TOTAL_AGENCIES:
                        self.__run_lottery()
                    send_done_ack(client_sock)
                    break  # cliente se reconecta para consultar

                elif msg_type == MSG_QUERY:
                    agency_id = recv_query(client_sock)
                    if not self._lottery_done:
                        send_not_ready(client_sock)
                    else:
                        send_winners(client_sock, self._winners.get(agency_id, []))
                    break

        except OSError as e:
            logging.error(f'action: receive_message | result: fail | error: {e}')
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
    

    def __run_lottery(self):
        for bet in load_bets():
            if has_won(bet):
                self._winners.setdefault(bet.agency, []).append(bet.document)
        self._lottery_done = True
        logging.info("action: sorteo | result: success")