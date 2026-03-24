import socket
import logging
import signal
from common.utils import Bet, store_bets, load_bets, has_won
from common.protocol import (
    recv_msg_type, recv_batch_payload, recv_done, recv_query,
    send_batch_ack, send_done_ack, send_winners, send_not_ready,
    MSG_BATCH, MSG_DONE, MSG_QUERY
)
from multiprocessing import Process, Barrier, Lock, Value, Manager


class Server:
    def __init__(self, port, listen_backlog, total_agencies):
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._running = True
        self._total_agencies = total_agencies
        self._barrier = Barrier(total_agencies)
        self._store_lock = Lock()
        self._lottery_done = Value('b', False)
        self._manager = Manager()
        self._winners = self._manager.dict()
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
                client_sock, addr = self.__accept_new_connection()
                logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
                p = Process(target=self.__handle_client_connection, args=(client_sock,))
                p.start()
                client_sock.close()
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
                    # ej8 update: protejo con lock ahora que es compartida entre procesos
                    with self._store_lock:
                        store_bets(bets)
                    logging.info(f'action: apuesta_recibida | result: success | cantidad: {len(bets)}')
                    send_batch_ack(client_sock, True)

                elif msg_type == MSG_DONE:
                    recv_done(client_sock)
                    self._barrier.wait()
                    self.__run_lottery()
                    send_done_ack(client_sock)
                    break  # cliente se reconecta para consultar

                elif msg_type == MSG_QUERY:
                    agency_id = recv_query(client_sock)
                    if not self._lottery_done.value:
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
        return c, addr

    def __run_lottery(self):
        with self._store_lock:
            if self._lottery_done.value:
                return  # otro proceso ya lo corrió
            for bet in load_bets():
                if has_won(bet):
                    current = self._winners.get(bet.agency, [])
                    self._winners[bet.agency] = current + [bet.document]
            self._lottery_done.value = True
            logging.info("action: sorteo | result: success")