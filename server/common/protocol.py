import struct

HEADER_SIZE = 2
STATUS_OK = b'\x00'
STATUS_ERROR = b'\x01'


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


def _decode_bet(data: bytes, offset: int):
    """Decodes a single bet from binary payload at offset.
    Returns (fields_dict, new_offset)"""
    agency = data[offset]
    offset += 1
    doc = struct.unpack_from('!I', data, offset)[0]
    offset += 4
    bd = struct.unpack_from('!I', data, offset)[0]
    offset += 4
    number = struct.unpack_from('!H', data, offset)[0]
    offset += 2

    fname_len = data[offset]
    offset += 1
    fname = data[offset:offset + fname_len].decode('utf-8')
    offset += fname_len

    lname_len = data[offset]
    offset += 1
    lname = data[offset:offset + lname_len].decode('utf-8')
    offset += lname_len

    year, month, day = bd // 10000, (bd % 10000) // 100, bd % 100
    birthdate = f"{year:04d}-{month:02d}-{day:02d}"

    return {
        'agency':     str(agency),
        'first_name': fname,
        'last_name':  lname,
        'document':   str(doc),
        'birthdate':  birthdate,
        'number':     str(number),
    }, offset


def recv_batch(sock):
    """Receives one batch. Returns list of bet dicts, or None on clean close.
    Frame: [2B payload_length][2B bet_count][bets...]"""
    try:
        header = recv_all(sock, HEADER_SIZE)
    except OSError:
        return None  # el cliente cerró la conexión gracefully
    length = struct.unpack('!H', header)[0]
    payload = recv_all(sock, length)

    count = struct.unpack_from('!H', payload, 0)[0]
    offset = 2
    bets = []
    for _ in range(count):
        bet, offset = _decode_bet(payload, offset)
        bets.append(bet)
    return bets


def send_batch_ack(sock, success: bool = True):
    """Sends 1-byte ACK: 0x00=OK, 0x01=ERROR"""
    send_all(sock, STATUS_OK if success else STATUS_ERROR)