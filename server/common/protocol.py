import struct

SEPARATOR = "|"
HEADER_SIZE = 2


def send_all(sock, data: bytes):
    """Sends all bytes avoiding short-write"""
    total = 0
    while total < len(data):
        sent = sock.send(data[total:])
        if sent == 0:
            raise OSError("Connection broken")
        total += sent


def recv_all(sock, n: int) -> bytes:
    """Reads exactly n bytes avoiding short-read"""
    data = b''
    while len(data) < n:
        chunk = sock.recv(n - len(data))
        if not chunk:
            raise OSError("Connection broken")
        data += chunk
    return data


def recv_bet(sock):
    """Receives a bet: reads 2-byte header then payload, returns field list"""
    header = recv_all(sock, HEADER_SIZE)
    length = struct.unpack('!H', header)[0]
    payload = recv_all(sock, length).decode('utf-8')
    # [agency, first_name, last_name, document, birthdate, number]
    return payload.split(SEPARATOR)


def send_confirmation(sock, msg: str = "OK"):
    """Sends a confirmation response using length-prefix framing"""
    data = msg.encode('utf-8')
    header = struct.pack('!H', len(data))
    send_all(sock, header + data)