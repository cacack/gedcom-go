"""Extract positioned text from a PDF, with no third-party dependency.

The GEDCOM 5.5 and 5.5.1 specifications exist only as PDFs, and their grammar
chapters are laid out in columns: a level marker, a tag, a value type, a
cardinality, and a page cross-reference, each drawn as a separately positioned
run. Reading them back as *lines* is the whole job -- a extractor that
concatenated runs in drawing order would interleave the cardinality column into
the middle of the tags.

So this walks each page's content stream, tracks the text matrix that PDF's
text operators maintain, and groups the shown strings by their y coordinate,
ordering each line by x. That is enough for these two documents. It is not a
general PDF text extractor and does not try to be: no font encoding tables (both
documents draw their body text in WinAnsi-compatible fonts), no ligature
handling, no right-to-left.

Used only by `transcribe.py`, which runs by hand. Nothing in the module or its
tests imports this.
"""

import re
import sys
import zlib

# A PDF object header: "12 0 obj".
OBJECT = re.compile(rb'(\d+)\s+(\d+)\s+obj\b')

# The stream that holds a page's drawing operators.
STREAM = re.compile(rb'stream\r?\n')

# A page's reference to the object holding its content stream.
CONTENTS = re.compile(rb'/Contents\s+(\d+)\s+\d+\s+R')

PAGE = re.compile(rb'/Type\s*/Page\b')

# One token of a content stream: a literal string, a hex string, a name, a
# number, an array bracket, or an operator.
TOKEN = re.compile(rb'''
    \((?:\\.|[^()\\])*\)      # literal string
  | <[0-9A-Fa-f\s]*>          # hex string
  | /[^\s/\[\]<>(){}]+        # name
  | [-+]?\d*\.?\d+            # number
  | \[ | \]
  | [A-Za-z'"*]+              # operator
''', re.S | re.X)

NUMBER = re.compile(rb'[-+]?\d*\.?\d+')

# Escapes PDF defines inside a literal string.
ESCAPES = {0x6e: 10, 0x72: 13, 0x74: 9, 0x62: 8, 0x66: 12,
           0x28: 40, 0x29: 41, 0x5c: 92}

# How far apart two runs must be, in text-space units, before the gap is read as
# a space. Below this they are parts of one word drawn separately.
GAP = 6

# How negative a TJ kerning adjustment must be to mean a space rather than
# tightening. A thousandth-of-an-em unit, so this is roughly a fifth of an em.
KERN_SPACE = -180


def _objects(data):
    """Split a PDF file into its numbered objects."""
    objects = {}
    for m in OBJECT.finditer(data):
        end = data.find(b'endobj', m.end())
        objects[int(m.group(1))] = data[m.end():end if end > 0 else len(data)]
    return objects


def _stream(obj):
    """Return an object's decoded stream, or None if it has none."""
    m = STREAM.search(obj)
    if not m:
        return None
    raw = obj[m.end():obj.find(b'endstream', m.end())]
    if b'FlateDecode' not in obj[:m.start()]:
        return raw
    try:
        return zlib.decompress(raw)
    except zlib.error:
        return None


def _unescape(s):
    """Resolve the escapes in a PDF literal string."""
    out = bytearray()
    i = 0
    while i < len(s):
        c = s[i]
        if c != 0x5c or i + 1 >= len(s):
            out.append(c)
            i += 1
            continue
        nxt = s[i + 1]
        if nxt in ESCAPES:
            out.append(ESCAPES[nxt])
            i += 2
        elif 0x30 <= nxt <= 0x37:
            digits = b''
            j = i + 1
            while j < len(s) and len(digits) < 3 and 0x30 <= s[j] <= 0x37:
                digits += bytes([s[j]])
                j += 1
            out.append(int(digits, 8) & 0xFF)
            i = j
        else:
            out.append(nxt)
            i += 2
    return bytes(out).decode('latin-1')


def _matmul(a, b):
    """Multiply two PDF 3x2 transformation matrices."""
    return [a[0] * b[0] + a[1] * b[2], a[0] * b[1] + a[1] * b[3],
            a[2] * b[0] + a[3] * b[2], a[2] * b[1] + a[3] * b[3],
            a[4] * b[0] + a[5] * b[2] + b[4], a[4] * b[1] + a[5] * b[3] + b[5]]


def _page_lines(content):
    """Read one page's content stream back as lines of text."""
    shown = []          # (y, x, order drawn, text)
    matrix = [1, 0, 0, 1, 0, 0]
    line_matrix = matrix[:]
    leading = 0.0
    operands = []

    def numbers(count):
        values = []
        for token in operands[-count:]:
            try:
                values.append(float(token))
            except ValueError:
                values.append(0.0)
        return values if len(values) == count else [0.0] * count

    def text_of(op):
        if op != b'TJ':
            for token in reversed(operands):
                if token[:1] == b'(':
                    return _unescape(token[1:-1])
            return ''
        # A TJ array interleaves strings with kerning adjustments; a large
        # negative adjustment is how these documents write a space.
        parts = []
        depth = 0
        for token in reversed(operands):
            if token == b']':
                depth += 1
                continue
            if token == b'[':
                depth -= 1
                if depth == 0:
                    break
                continue
            parts.append(token)
        text = ''
        for token in reversed(parts):
            if token[:1] == b'(':
                text += _unescape(token[1:-1])
            elif NUMBER.fullmatch(token) and float(token) < KERN_SPACE:
                text += ' '
        return text

    for match in TOKEN.finditer(content):
        token = match.group()
        if (token[:1] in (b'(', b'<', b'/') or token in (b'[', b']')
                or NUMBER.fullmatch(token)):
            operands.append(token)
            continue

        op = token
        if op == b'BT':
            matrix = [1, 0, 0, 1, 0, 0]
            line_matrix = matrix[:]
        elif op == b'Tm':
            matrix = numbers(6)
            line_matrix = matrix[:]
        elif op in (b'Td', b'TD'):
            tx, ty = numbers(2)
            if op == b'TD':
                leading = -ty
            line_matrix = _matmul([1, 0, 0, 1, tx, ty], line_matrix)
            matrix = line_matrix[:]
        elif op == b'TL':
            leading = numbers(1)[0]
        elif op == b'T*':
            line_matrix = _matmul([1, 0, 0, 1, 0, -leading], line_matrix)
            matrix = line_matrix[:]
        elif op in (b'Tj', b'TJ', b"'", b'"'):
            if op in (b"'", b'"'):
                line_matrix = _matmul([1, 0, 0, 1, 0, -leading], line_matrix)
                matrix = line_matrix[:]
            text = text_of(op)
            if text:
                shown.append((round(matrix[5]), matrix[4], len(shown), text))
        operands = []

    rows = {}
    for y, x, order, text in shown:
        rows.setdefault(y, []).append((x, order, text))

    lines = []
    for y in sorted(rows, reverse=True):
        # Ties on x keep the order the runs were drawn in: consecutive show
        # operators share a position, and sorting those by their text would
        # reverse words within a line.
        buffer = ''
        end_of_previous = None
        for x, _, text in sorted(rows[y]):
            if end_of_previous is not None and x - end_of_previous > GAP:
                buffer += ' '
            buffer += text
            # No font widths are read, so a run's extent is estimated. It only
            # has to be good enough to tell a word gap from a kerned join.
            end_of_previous = x + 5 * len(text)
        lines.append(buffer.rstrip())
    return lines


def extract(path):
    """Return the text of every page of a PDF, as a list of lines."""
    with open(path, 'rb') as f:
        data = f.read()

    objects = _objects(data)
    lines = []
    for number, obj in sorted(objects.items()):
        if not PAGE.search(obj):
            continue
        contents = CONTENTS.search(obj)
        if not contents:
            continue
        stream = _stream(objects.get(int(contents.group(1)), b''))
        if stream is None:
            continue
        lines.append(f'=== page (obj {number}) ===')
        lines.extend(_page_lines(stream))
    return lines


if __name__ == '__main__':
    print('\n'.join(extract(sys.argv[1])))
