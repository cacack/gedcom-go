"""Parse the Lineage-Linked Grammar chapters of the GEDCOM 5.5/5.5.1 specifications.

The grammar is not published in machine-readable form, so this reads it out of
the specification PDFs and turns it into the shape the 7.0 standards body
publishes directly: (superstructure, tag, structure) triples, with cardinalities
and payload types alongside.

The notation being read looks like this:

    HEADER: =
    n HEAD                                {1:1}
      +1 SOUR <APPROVED_SYSTEM_ID>        {1:1} p.42
        +2 CORP <NAME_OF_BUSINESS>        {0:1} p.54
          +3 <<ADDRESS_STRUCTURE>>        {0:1} p.31

Four constructs carry all the meaning:

  * a *block* (``HEADER: =``) names a group of lines;
  * ``n`` and ``+k`` give nesting, relative to the block's own base level;
  * ``<<NAME>>`` splices another block in at that level -- the indirection that
    makes a flat transcription impossible;
  * ``[ A | B ]`` alternates, either over tags on one line or over whole
    branches across lines.

A structure is identified by where the grammar *defines* it, not by how a
document reaches it: ``EVENT_DETAIL.DATE`` is one structure however many event
tags splice EVENT_DETAIL in. That is what makes the output a graph rather than a
tree, and it is the same identity the 7.0 files use when they give one URI a
dozen superstructure rows.

Nothing here corrects the specification. Where the text is defective -- and both
documents are, in small ways -- the defect and the reading taken are recorded,
so the transcription produces the list a reviewer needs rather than a silently
repaired grammar.
"""

import re
from dataclasses import dataclass, field

# Where the grammar chapter starts. It opens with the whole-file production and
# closes where the primitive (value type) definitions begin; those are
# recognizable by the {Size=lo:hi} annotation, which no structure line carries.
# Both section headings inside the chapter are set in a subsetted font that
# extracts as mojibake, so headings cannot be matched on.
GRAMMAR_START = re.compile(r'^LINEAGE_LINKED_GEDCOM\s*:\s*=\s*$')

# A block header: a production name alone on its line.
BLOCK = re.compile(r'^([A-Z][A-Z0-9_]*)\s*:\s*=\s*$')

# A primitive definition, with its recommended field size. Also the marker for
# the end of the grammar chapter.
# A primitive definition, with its recommended field size. Also the marker for
# the end of the grammar chapter. Both documents spell the annotation
# irregularly in places -- one uses "SIZE", and four state a single width
# rather than a range -- so both are accepted and recorded.
PRIMITIVE = re.compile(
    r'^([A-Z][A-Z0-9_]*)\s*:\s*=\s*\{\s*(Size|SIZE)\s*=\s*(\d+)\s*(?::\s*(\d+)\s*)?\}\s*$')

# An enumeration of literal values, as some primitives are defined by. Only the
# single-line form is read; the multi-line ones (AGE_AT_EVENT) describe a syntax
# rather than a value set, and the harness knows how to build those.
ENUM = re.compile(r'^\[\s*([A-Za-z0-9_]+(?:\s*\|\s*[A-Za-z0-9_]+)+)\s*\]\s*$')

# The level marker opening a grammar line. "n" is the block's own base level,
# "+k" is k levels below it, and "0" is the absolute level the whole-file
# production is written at. Both documents lose the space after the marker in
# places ("nEVEN", "+1AGE"), so the space is optional -- but then what follows
# has to be something a tag line can start with, or every prose sentence
# beginning "number" or "not" would read as a level marker.
LEVEL = re.compile(r'^(?:(n)|\+(\d+)|(0))(?:\s+|(?=[A-Z_@<\[]))')

# Cardinality. The closing brace is wrong in several places across the two
# documents -- ")", "]", or nothing at all where a "*" footnote marker ran into
# it -- so anything that terminates the range is accepted and the defect is
# recorded.
CARDINALITY = re.compile(r'\{\s*(\d+)\s*:\s*(\d+|M)\s*(\*?)\s*(\}|\)|\]|\Z)')

# Trailing page cross-references: "p.42", "p.63, 64", "p.33,26", "p.42/41".
PAGEREF = re.compile(r'\s*p\.\s*\d+(?:\s*[,/]\s*\d+)*\s*$')

# An inline comment.
COMMENT = re.compile(r'/\*.*?\*/')

# A substructure splice, tolerating the stray inner space the text has in
# several places ("<<ADDRESS_STRUCTURE >>").
REFERENCE = re.compile(r'^<<\s*([A-Z][A-Z0-9_]*)\s*>>$')

# A cross-reference, in the bracketed form and in the unbracketed form the text
# also uses ("@<XREF:FAM>@" and "@XREF:FAM@"), with the stray space 5.5.1's
# MULTIMEDIA_LINK has ("@<XREF:OBJE> @").
POINTER = re.compile(r'^@\s*<?\s*XREF\s*:\s*([A-Z]+)\s*>?\s*@')

# An alternation over tags on one line: "[ ANUL | CENS | DIV | DIVF ]".
TAG_ALTERNATION = re.compile(r'^\[\s*([A-Z][A-Z0-9]*(?:\s*\|\s*[A-Z][A-Z0-9]*)+)\s*\]')

# A bare tag. Four characters in the standard; 5.5.1 adds the five-character
# EMAIL. The leading underscore is the extension form, which neither
# specification uses but which Gedcom.pm's transcriptions add, so accepting it
# keeps the cross-check comparing grammars rather than parser limitations.
TAG = re.compile(r'^(_[A-Z][A-Z0-9_]*|[A-Z][A-Z0-9]{1,4})(?![A-Za-z0-9_])')


@dataclass
class Defect:
    """A place where the specification text does not say what it means."""

    block: str
    line: str
    problem: str
    reading: str


@dataclass
class Entry:
    """One line of the grammar: a tag, or a splice of another block."""

    level: int
    cardinality: str = ''
    # Set on a tag line.
    tag: str = ''
    structure: str = ''
    payload: str = ''
    xref: str = ''
    # Set on a "<<NAME>>" line instead.
    reference: str = ''
    children: list = field(default_factory=list)


def multiply(outer, inner, exclusive=False):
    """Compose a splice's cardinality with the one written inside the block.

    A block spliced in {0:M} times, each occurrence allowing {1:1} of a tag,
    permits {0:M} of that tag; one spliced {0:1} time whose block allows {0:3}
    permits {0:3}. The effective range is the product.

    Except across an alternation, where the branches are mutually exclusive and
    a product would state each one as required. RECORD is spliced {1:M} and its
    seven branches are each {1:1}, which multiplies to {1:M} per record type --
    reading as though a conformant file must contain a family *and* an
    individual *and* a repository. What {1:M} requires is one of the set, so an
    exclusive branch keeps the upper bound and takes a lower bound of zero.
    """
    if not outer:
        return inner
    if not inner:
        return outer
    outer_lo, outer_hi = outer.split(':')
    inner_lo, inner_hi = inner.split(':')
    lo = 0 if exclusive else int(outer_lo) * int(inner_lo)
    hi = 'M' if 'M' in (outer_hi, inner_hi) else str(int(outer_hi) * int(inner_hi))
    return f'{lo}:{hi}'


class Grammar:
    """The parsed grammar chapter of one specification."""

    def __init__(self):
        # Production name -> the entries written at that block's base level.
        self.blocks = {}
        # Structure identity -> the entry that defines it.
        self.structures = {}
        # Primitive name -> {'size': 'lo:hi', 'values': [...]}.
        self.primitives = {}
        # Blocks whose base-level entries are branches of an alternation, and
        # so are alternatives to each other rather than siblings.
        self.alternations = set()
        self.defects = []
        self.unparsed = []
        self._top = {}
        self._extra = {}

    def defect(self, block, line, problem, reading):
        self.defects.append(Defect(block, line, problem, reading))

    # -- the structure graph ----------------------------------------------

    def roots(self):
        """The structures a document may contain at level 0."""
        return self.top('LINEAGE_LINKED_GEDCOM')

    def top(self, block):
        """The (tag, structure, cardinality) triples a block contributes.

        Splices resolve to the referenced block's own top-level triples, so this
        is the block's outermost surface -- what appears where it is spliced in.
        """
        if block not in self._top:
            if block not in self.blocks:
                raise KeyError(f'no block named {block}')
            self._top[block] = self._resolve(self.blocks[block], block)
        return self._top[block]

    def children(self, structure):
        """The (tag, structure, cardinality) triples nested under a structure."""
        entry = self.structures[structure]
        return self._resolve(entry.children + self._extra.get(structure, []), structure)

    def _resolve(self, entries, where):
        """Expand one level of entries into the triples they contribute.

        This recurses through splices but never into children, so it is bounded
        by the block graph's outermost surface even though the structure graph
        itself has cycles (5.5's NOTE_STRUCTURE and SOURCE_CITATION splice each
        other, so a note is reachable from within a note).
        """
        out = []
        for entry in entries:
            if not entry.reference:
                out.append((entry.tag, entry.structure, entry.cardinality))
                continue
            exclusive = entry.reference in self.alternations
            for tag, structure, cardinality in self.top(entry.reference):
                out.append((tag, structure,
                            multiply(entry.cardinality, cardinality, exclusive)))
        return out

    def collect_extras(self):
        """Attach lines written *under* a splice to what the splice contributes.

        5.5's FAM_RECORD nests HUSB and WIFE beneath <<FAMILY_EVENT_STRUCTURE>>,
        meaning they are substructures of every family event tag rather than of
        the family itself. 5.5.1 later gave that reading a name --
        FAMILY_EVENT_DETAIL, holding exactly those two tags -- which is what
        confirms it.

        The attachment is global to the spliced structure rather than scoped to
        the splice site; `verify` fails if a block that needs it is ever spliced
        from two places, which is what would make the difference visible.
        """
        for block, entries in self.blocks.items():
            for entry in _walk(entries):
                if entry.reference and entry.children:
                    for _, structure, _ in self.top(entry.reference):
                        self._extra.setdefault(structure, []).extend(entry.children)

    # -- verification ------------------------------------------------------

    def verify(self):
        """Report violations of the assumptions the rest of this rests on."""
        problems = []

        for block, entries in self.blocks.items():
            if not entries:
                problems.append(f'block {block} has no grammar lines')

        sites = {}
        needs_scope = set()
        for block, entries in self.blocks.items():
            for entry in _walk(entries):
                if not entry.reference:
                    continue
                sites.setdefault(entry.reference, []).append(block)
                if entry.children:
                    needs_scope.add(entry.reference)
                if entry.reference not in self.blocks:
                    problems.append(f'{block} splices <<{entry.reference}>>, which is not defined')

        for block in sorted(needs_scope):
            if len(sites.get(block, [])) > 1:
                problems.append(
                    f'{block} is spliced from {len(sites[block])} places and has lines '
                    'nested under a splice; collect_extras attaches those globally')

        for block in sorted(self.blocks):
            if block != 'LINEAGE_LINKED_GEDCOM' and block not in sites:
                problems.append(f'block {block} is defined but never spliced')

        return problems


def _walk(entries):
    for entry in entries:
        yield entry
        yield from _walk(entry.children)


def _clean(lines):
    """Drop what the page carries that the grammar does not."""
    out = []
    for line in lines:
        if line.startswith('=== page '):
            continue
        if re.fullmatch(r'\s*\d+\s*', line):  # a page number, alone on its line
            continue
        if line.strip():
            out.append(line.strip())
    return out


def _structural(line):
    """Whether the line is a branch boundary of a multi-line alternation."""
    bare = COMMENT.sub('', line).strip()
    return bare in ('[', '|', ']')


def parse(lines):
    """Parse the grammar chapter out of the extracted specification text."""
    g = Grammar()
    lines = _clean(lines)

    start = next((i for i, line in enumerate(lines) if GRAMMAR_START.match(line)), None)
    if start is None:
        raise SystemExit('the LINEAGE_LINKED_GEDCOM production was not found')

    block = 'LINEAGE_LINKED_GEDCOM'
    g.blocks[block] = []
    # (level, the list new entries at that level go into, the tag path to it).
    stack = [(-1, g.blocks[block], [])]
    started = False  # whether this block has produced a grammar line yet
    closed = False   # whether prose has since ended it
    depth = 0        # open alternation brackets in the current block
    counts = {}      # structure identity -> how many times it has been used

    i = start + 1
    while i < len(lines):
        line = lines[i]
        i += 1

        if PRIMITIVE.match(line):
            i -= 1
            break

        header = BLOCK.match(line)
        if header:
            if depth:
                g.defect(block, '[', 'an alternation is opened and never closed',
                         'read as closing at the end of the block, which is where '
                         'its last branch ends anyway')
            depth = 0
            block = header.group(1)
            g.blocks[block] = []
            stack = [(-1, g.blocks[block], [])]
            started, closed = False, False
            continue

        if _structural(line):
            # Every branch of an alternation contributes its lines at the base
            # level, so the only effect of a boundary is to return there.
            bare = COMMENT.sub('', line).strip()
            if bare == '|' and depth == 0:
                g.defect(block, '|', 'a branch separator appears outside any "[ ... ]"',
                         'read as an alternation spanning the whole block')
            g.alternations.add(block)
            depth += {'[': 1, ']': -1}.get(bare, 0)
            stack = stack[:1]
            continue

        if closed or not LEVEL.match(line):
            # Prose. Before the block's first grammar line it is a preamble;
            # after it, it ends the block. That is what keeps 5.5.1's "some
            # systems may have output the following 5.5 structure" illustration
            # -- written in grammar notation, inside MULTIMEDIA_LINK's prose --
            # out of the grammar.
            closed = started
            continue

        entries = _parse_line(g, block, line, stack, counts)
        if entries is None:
            g.unparsed.append((block, line))
            continue
        started = True

        level = entries[0].level
        while len(stack) > 1 and stack[-1][0] >= level:
            stack.pop()
        parent_path = stack[-1][2]
        stack[-1][1].extend(entries)

        # Alternated tags on one line share whatever nests under them, so they
        # share one child list; the path follows the first, which is the one the
        # specification's own cross-references name.
        shared = entries[0].children
        for entry in entries[1:]:
            entry.children = shared
        segment = entries[0].reference or entries[0].structure.rsplit('.', 1)[-1]
        stack.append((level, shared, parent_path + [segment]))

    if depth:
        g.defect(block, '[', 'an alternation is opened and never closed',
                 'read as closing at the end of the block, which is where '
                 'its last branch ends anyway')

    _parse_primitives(g, lines[i:])
    return g


def _identity(block, path, counts):
    """Name a structure by where the grammar defines it.

    The name is the block plus the path of tags down to the line, so two blocks
    defining the same tag stay distinct while one block reached from ten places
    stays single. Alternation branches can repeat a tag within a block -- both
    documents define NOTE twice in NOTE_STRUCTURE, once as a pointer and once
    inline -- so a repeat is suffixed rather than merged: they are different
    structures, and this library may well support one and not the other.
    """
    name = '.'.join([block] + path)
    counts[name] = counts.get(name, 0) + 1
    return name if counts[name] == 1 else f'{name}#{counts[name]}'


def _parse_line(g, block, line, stack, counts):
    """Parse one grammar line into the entries it defines, or None."""
    marker = LEVEL.match(line)
    if not marker:
        return None
    level = 0 if marker.group(1) or marker.group(3) else int(marker.group(2))
    body = line[marker.end():]

    body = PAGEREF.sub('', body)
    body = COMMENT.sub(' ', body).strip()

    cardinality = ''
    matches = list(CARDINALITY.finditer(body))
    if matches:
        card = matches[-1]
        cardinality = f'{card.group(1)}:{card.group(2)}'
        if card.group(3):
            g.defect(block, line, 'a footnote marker sits inside the cardinality braces',
                     f'read as {{{cardinality}}}')
        if card.group(4) in (')', ']'):
            g.defect(block, line, f'the cardinality closes with "{card.group(4)}", not "}}"',
                     f'read as {{{cardinality}}}')
        elif card.group(4) == '':
            g.defect(block, line, 'the cardinality is never closed',
                     f'read as {{{cardinality}}}')
        body = body[:card.start()].strip()
    else:
        g.defect(block, line, 'the line states no cardinality',
                 'read as the cardinality of the splice that reaches it, which is '
                 'the only bound stated for it anywhere')

    reference = REFERENCE.match(body)
    if reference:
        if reference.group(1) not in ('LINEAGE_LINKED_GEDCOM',) and ' ' in reference.group(0):
            g.defect(block, line, 'the substructure name has a stray space before ">>"',
                     f'read as <<{reference.group(1)}>>')
        return [Entry(level=level, cardinality=cardinality, reference=reference.group(1))]

    xref = ''
    pointer = POINTER.match(body)
    if pointer:
        # A leading cross-reference is the record's own identifier, not a
        # payload: "n @<XREF:FAM>@ FAM". The tag follows it.
        xref = pointer.group(1)
        if '<' not in pointer.group(0):
            g.defect(block, line, 'the cross-reference is written without its angle brackets',
                     f'read as @<XREF:{xref}>@')
        body = body[pointer.end():].strip()

    alternation = TAG_ALTERNATION.match(body)
    if alternation:
        tags = [t.strip() for t in alternation.group(1).split('|')]
        payload = body[alternation.end():].strip()
    else:
        tag = TAG.match(body)
        if not tag:
            return None
        tags = [tag.group(1)]
        payload = body[tag.end():].strip()

    payload = _normalize_payload(g, block, line, re.sub(r'\s+', ' ', payload))

    entries = []
    for tag in tags:
        path = _path_for(stack, level, tag)
        entries.append(Entry(
            level=level,
            cardinality=cardinality,
            tag=tag,
            structure=_identity(block, path, counts),
            payload=payload,
            xref=xref,
        ))
    for entry in entries:
        g.structures[entry.structure] = entry
    return entries


def _path_for(stack, level, tag):
    """The tag path a line at this level sits on, given the open stack."""
    path = []
    for entry_level, _, entry_path in stack:
        if entry_level < level:
            path = entry_path
    return path + [tag]


def _normalize_payload(g, block, line, payload):
    """Regularize a payload's spelling, recording what had to be regularized.

    Only whitespace and the cross-reference brackets are touched. A value type
    the text names wrongly is left exactly as written and reported: guessing at
    what it meant to say is the one thing this must not do.
    """
    if not payload:
        return payload

    if payload.count('<') != payload.count('>'):
        g.defect(block, line, 'the value type is missing an angle bracket',
                 f'read as the value type named in "{payload}"')

    def canonical(match):
        text = match.group(0)
        fixed = f'@<XREF:{match.group(1)}>@'
        if text != fixed:
            g.defect(block, line, 'the cross-reference payload is spelled irregularly',
                     f'read as {fixed}')
        return fixed

    return re.sub(r'@\s*<?\s*XREF\s*:\s*([A-Z]+)\s*>?\s*@', canonical, payload)


def _parse_primitives(g, lines):
    """Read the primitive (value type) definitions that follow the grammar."""
    name = None
    for line in lines:
        header = PRIMITIVE.match(line)
        if header:
            name = header.group(1)
            if header.group(2) != 'Size':
                g.defect(name, line, 'the size annotation is spelled "SIZE", not "Size"',
                         'read as a size annotation')
            if header.group(4) is None:
                g.defect(name, line, 'the size annotation states one width, not a range',
                         f'read as a maximum of {header.group(3)} with no minimum')
            low, high = (header.group(3), header.group(4)) if header.group(4) else ('', header.group(3))
            g.primitives[name] = {'size': f'{low}:{high}', 'values': []}
            continue
        if name is None:
            continue
        enum = ENUM.match(line)
        if enum and not g.primitives[name]['values']:
            g.primitives[name]['values'] = [v.strip() for v in enum.group(1).split('|')]
        elif not enum and not line.startswith('['):
            # Only the enumeration immediately following the header belongs to
            # it; prose after that describes the values rather than listing more.
            name = None
