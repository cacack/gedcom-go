#!/usr/bin/env python3
"""Cross-check the transcribed 5.5/5.5.1 inventories against independent sources.

A transcription of a printed grammar is only as good as its proofreading, so it
is checked twice, against sources that were made without reference to it:

  1. **Appendix B of the 5.5 specification** -- the document's own Structure
     Cross-Reference, listing every place each substructure is spliced. That
     makes the 5.5 document self-checking: the grammar chapter and the appendix
     were typeset from the same source and either agrees with our reading of the
     grammar or does not. 5.5.1 dropped the appendix, so this check runs for 5.5
     only.

  2. **pjcj/Gedcom.pm's `.grammar` files** -- transcriptions of these same
     chapters by someone else, whose README notes they "had to modify slightly"
     the standard versions "to correct a few errors". They are written in the
     specification's own notation, so the parser used on the PDFs reads them
     directly, and the two inventories can be compared as data.

Gedcom.pm is Perl-licensed (Artistic/GPL), which this MIT repository cannot
vendor. Nothing is copied: the files are fetched to a temporary path, read, and
the differences reported. What lands in the repository is our own transcription
and the list of places the two disagree.

Usage:

    python3 scripts/spec55/crosscheck.py ged55.pdf ged551.pdf [--offline=DIR]

Every difference is expected to be explainable. The point is not that the two
agree everywhere -- they should not, since Gedcom.pm departs from the printed
grammar in places and this follows it -- but that each disagreement has a reason
recorded against it, in testdata/spec/README.md.
"""

import os
import re
import sys
import urllib.request

import grammar
import pdftext
import transcribe

# Gedcom.pm's transcriptions of the same two chapters.
GEDCOM_PM = {
    '5.5': 'https://raw.githubusercontent.com/pjcj/Gedcom.pm/master/gedcom-5.5.grammar',
    '5.5.1': 'https://raw.githubusercontent.com/pjcj/Gedcom.pm/master/gedcom-5.5.1.grammar',
}

# Gedcom.pm names the whole-file production GEDCOM rather than
# LINEAGE_LINKED_GEDCOM. Renaming it is the only edit made to their text, and it
# is made in memory.
PM_START = re.compile(r'^\s*GEDCOM\s*:\s*=\s*$')

# Appendix B of the 5.5 specification: "<<NOTE_STRUCTURE>> FAM_RECORD:= p. 24".
APPENDIX_B = re.compile(r'^<<([A-Z][A-Z0-9_]*)>>\s+([A-Z][A-Z0-9_]*)\s*:\s*=')

APPENDIX_B_HEADING = 'Structure Cross-Reference'

# Where Appendix B's own listing stops.
PRIMITIVE_HEADING = 'Primitive Cross-Reference'


def splice_sites(g):
    """Every (spliced block, block it is spliced into) pair in a grammar."""
    sites = set()
    for block, entries in g.blocks.items():
        for entry in grammar._walk(entries):
            if entry.reference:
                sites.add((entry.reference, block))
    return sites


def appendix_b(text):
    """The splice sites the 5.5 specification lists for itself."""
    sites = set()
    reading = False
    for line in text:
        stripped = line.strip()
        if stripped.startswith(APPENDIX_B_HEADING):
            reading = True
            continue
        if stripped.startswith(PRIMITIVE_HEADING):
            reading = False
            continue
        if not reading:
            continue
        m = APPENDIX_B.match(stripped)
        if m:
            sites.add((m.group(1), m.group(2)))
    return sites


def gedcom_pm(version, offline):
    """Parse Gedcom.pm's transcription of one version into our own shape."""
    if offline:
        path = os.path.join(offline, f'gedcom-{version}.grammar')
        with open(path, encoding='utf-8', errors='replace') as f:
            text = f.read()
    else:
        with urllib.request.urlopen(GEDCOM_PM[version], timeout=60) as response:
            text = response.read().decode('utf-8', errors='replace')

    lines = [PM_START.sub('LINEAGE_LINKED_GEDCOM: =', line) for line in text.splitlines()]
    g = grammar.parse(lines)
    g.collect_extras()
    if g.unparsed:
        # A line their transcription writes and this parser cannot read would
        # show up below as a difference in the grammars, which it is not.
        print(f'\n== {version}: lines of Gedcom.pm this parser could not read')
        for block, line in g.unparsed:
            print(f'   {block}: {line}')
    return g


def pairs(g):
    """The (superstructure, tag) set an inventory describes.

    Both transcriptions keep the specification's own production names, so the
    structure identities this parser derives are comparable between them, and
    comparing on those rather than on parent tags alone is what catches a tag
    added under one SOUR and not the other.

    One class of difference is an artifact rather than a disagreement. Where a
    line alternates tags -- "n [ BAPL | CONL ]" -- the branches share one child
    list here and one identity is derived for it, from the first tag; a
    transcription that writes the branches out separately derives two. The
    substructures are identical either way, and only the name differs.
    """
    return {(superstructure, tag)
            for superstructure, tag, _, _ in transcribe.inventory(g)}


def report(title, ours, theirs, us, them):
    print(f'\n== {title}')
    only_ours = sorted(ours - theirs)
    only_theirs = sorted(theirs - ours)
    print(f'   {len(ours & theirs)} agree, {len(only_ours)} only in {us}, '
          f'{len(only_theirs)} only in {them}')
    for item in only_ours:
        print(f'   only in {us}:   {" ".join(item)}')
    for item in only_theirs:
        print(f'   only in {them}: {" ".join(item)}')
    return len(only_ours) + len(only_theirs)


def main(argv):
    args = [a for a in argv[1:] if not a.startswith('--')]
    offline = next((a.split('=', 1)[1] for a in argv[1:] if a.startswith('--offline=')), None)
    if len(args) != 2:
        raise SystemExit('usage: crosscheck.py ged55.pdf ged551.pdf [--offline=DIR]')

    differences = 0
    for pdf, edition in zip(args, transcribe.EDITIONS):
        version = edition['version']
        text = pdftext.extract(pdf)
        ours = grammar.parse(text)
        ours.collect_extras()

        if version == '5.5':
            listed = appendix_b(text)
            differences += report(
                f'{version}: splice sites, grammar chapter vs Appendix B',
                splice_sites(ours), listed, 'the grammar chapter', 'Appendix B')

        theirs = gedcom_pm(version, offline)
        differences += report(
            f'{version}: (superstructure, tag) pairs, ours vs Gedcom.pm',
            pairs(ours), pairs(theirs), 'ours', 'Gedcom.pm')

    print(f'\n{differences} differences to account for')


if __name__ == '__main__':
    main(sys.argv)
