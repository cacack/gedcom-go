#!/usr/bin/env python3
"""Transcribe the GEDCOM 5.5 and 5.5.1 grammars into machine-readable spec files.

GEDCOM 7.0 is published with an ``extracted-files/`` directory: the standards
body's own machine-readable statement of what the specification defines. 5.5 and
5.5.1 have no such thing -- their structure set exists only as the semi-formal
Lineage-Linked Grammar printed in Chapter 2 of each PDF. This reads those
chapters and writes the same files 7.0 ships, so that one coverage harness can
measure all three versions against the same shape.

Usage:

    python3 scripts/spec55/transcribe.py ged55.pdf ged551.pdf

The PDFs are not vendored, and neither is their prose. What lands in the
repository is the *facts* read out of them -- tag names, nesting, cardinality,
value types -- plus the individual grammar lines `defects.tsv` has to quote to
say what is wrong with them. Each document states its own terms, which SOURCE
records verbatim rather than paraphrasing, alongside the URL and SHA-256 of the
file it was read from, so a reviewer can re-run this against the same bytes.

What comes out, per version, under testdata/spec/:

    substructures.tsv   superstructure, tag, structure
    cardinalities.tsv   superstructure, structure, cardinality
    payloads.tsv        structure, payload
    primitives.tsv      primitive, size, values
    defects.tsv         block, line, problem, reading
    SOURCE              provenance of the PDF this was read from

`decoder/spec55_coverage_test.go` consumes them. Nothing at runtime does.
"""

import hashlib
import os
import re
import sys

import grammar
import pdftext

HERE = os.path.dirname(os.path.abspath(__file__))
TESTDATA = os.path.join(HERE, '..', '..', 'testdata', 'spec')

# The specification each PDF is expected to be, and where it came from. The
# release each document states of itself is checked, so passing the two files in
# the wrong order fails rather than transcribing 5.5 as 5.5.1.
EDITIONS = [
    {
        'version': '5.5',
        'directory': 'gedcom-5.5',
        'url': 'https://gedcom.io/specifications/ged55.pdf',
    },
    {
        'version': '5.5.1',
        'directory': 'gedcom-5.5.1',
        'url': 'https://gedcom.io/specifications/ged551.pdf',
    },
]

# The release statement on the title page, which both documents carry -- though
# not on the first page they lay out, so the whole file is searched.
RELEASE = re.compile(r'^Release\s+([\d.]+)$')


# The copyright notice, which the two documents word differently: 5.5 grants a
# limited permission to copy and 5.5.1 states the bare reservation. Reading it
# out of the document rather than restating it is the point -- one constant for
# both editions is how the wrong years got recorded for 5.5.
COPYRIGHT = re.compile(r'^Copyright\b.*Latter-day Saints')


def stated_copyright(text):
    """The copyright notice a specification document carries, verbatim.

    Returned as one line: the notice wraps across two in 5.5, where the
    permission sentence follows the reservation, so lines are joined until the
    sentence ends.
    """
    for i, line in enumerate(text):
        if not COPYRIGHT.match(line.strip()):
            continue
        notice = line.strip()
        for continuation in text[i + 1:i + 4]:
            if notice.endswith('.'):
                break
            notice += ' ' + continuation.strip()
        return ' '.join(notice.split())
    return None


def stated_release(text):
    """The release a specification document says it is."""
    for line in text:
        m = RELEASE.match(' '.join(line.split()))
        if m:
            return m.group(1)
    return None


def inventory(g):
    """Walk the structure graph into (superstructure, tag, structure) triples.

    Breadth-first from the level-0 structures, visiting each structure once. The
    graph has cycles -- 5.5 lets a note cite a source that carries a note -- so
    the visited set is what makes this terminate, and it is also what makes a
    structure reached ten ways get counted once.
    """
    rows = []
    seen = set()
    queue = []

    for tag, structure, cardinality in g.roots():
        rows.append(('', tag, structure, cardinality))
        if structure not in seen:
            seen.add(structure)
            queue.append(structure)

    while queue:
        current = queue.pop(0)
        for tag, structure, cardinality in g.children(current):
            rows.append((current, tag, structure, cardinality))
            if structure not in seen:
                seen.add(structure)
                queue.append(structure)

    # Sorted rather than left in walk order: the walk order depends on which
    # path reached a structure first, and a stable file makes the diff of a
    # re-transcription readable.
    return sorted(set(rows))


def write_tsv(path, header, rows):
    with open(path, 'w', encoding='utf-8', newline='\n') as f:
        f.write('\t'.join(header) + '\n')
        for row in rows:
            f.write('\t'.join(row) + '\n')


def transcribe(pdf, edition):
    text = pdftext.extract(pdf)
    stated = stated_release(text)
    if stated != edition['version']:
        raise SystemExit(
            f'{pdf} states it is release {stated}, not {edition["version"]}; '
            'pass ged55.pdf first and ged551.pdf second')

    notice = stated_copyright(text)
    if notice is None:
        raise SystemExit(f'{pdf} carries no copyright notice this can read; SOURCE '
                         'records it verbatim rather than restating it')

    g = grammar.parse(text)
    g.collect_extras()

    problems = g.verify()
    if problems:
        raise SystemExit('the parsed grammar is not usable:\n  ' + '\n  '.join(problems))
    if g.unparsed:
        raise SystemExit('grammar lines were not understood:\n  ' + '\n  '.join(
            f'{block}: {line}' for block, line in g.unparsed))

    rows = inventory(g)
    directory = os.path.join(TESTDATA, edition['directory'])
    os.makedirs(directory, exist_ok=True)

    write_tsv(os.path.join(directory, 'substructures.tsv'),
              ('superstructure', 'tag', 'structure'),
              [(sup, tag, structure) for sup, tag, structure, _ in rows])

    write_tsv(os.path.join(directory, 'cardinalities.tsv'),
              ('superstructure', 'structure', 'cardinality'),
              sorted({(sup, structure, card) for sup, _, structure, card in rows}))

    used = sorted({structure for _, _, structure, _ in rows})
    write_tsv(os.path.join(directory, 'payloads.tsv'),
              ('structure', 'payload'),
              [(s, g.structures[s].payload) for s in used])

    write_tsv(os.path.join(directory, 'primitives.tsv'),
              ('primitive', 'size', 'values'),
              [(name, p['size'], '|'.join(p['values']))
               for name, p in sorted(g.primitives.items())])

    write_tsv(os.path.join(directory, 'defects.tsv'),
              ('block', 'line', 'problem', 'reading'),
              sorted({(d.block, ' '.join(d.line.split()), d.problem, d.reading)
                      for d in g.defects}))

    with open(os.path.join(directory, 'SOURCE'), 'w', encoding='utf-8', newline='\n') as f:
        f.write(f'document: The GEDCOM Standard Release {edition["version"]}\n')
        f.write(f'url: {edition["url"]}\n')
        f.write(f'sha256: {sha256(pdf)}\n')
        f.write(f'retrieved: {os.environ.get("SPEC55_RETRIEVED", "unknown")}\n')
        f.write(f'copyright: {notice}\n')
        f.write('transcribed-by: scripts/spec55/transcribe.py\n')

    return len(rows), len(used), len(g.defects)


def sha256(path):
    h = hashlib.sha256()
    with open(path, 'rb') as f:
        for chunk in iter(lambda: f.read(1 << 16), b''):
            h.update(chunk)
    return h.hexdigest()


def main(argv):
    if len(argv) != 3:
        raise SystemExit(__doc__.strip().splitlines()[0] +
                         '\n\nusage: transcribe.py ged55.pdf ged551.pdf')
    if 'SPEC55_RETRIEVED' not in os.environ:
        raise SystemExit('set SPEC55_RETRIEVED=YYYY-MM-DD to the date the PDFs were '
                         'downloaded; it is recorded in SOURCE')
    for pdf, edition in zip(argv[1:], EDITIONS):
        pairs, structures, defects = transcribe(pdf, edition)
        print(f'{edition["version"]}: {pairs} pairs, {structures} structures, '
              f'{defects} specification defects recorded')


if __name__ == '__main__':
    main(sys.argv)
