So far, these files have been added in an ad-hoc manner to test a 5.5.1 to 7.0 converter.

Note that there is more than one way to express the same data, and hence more than one way to convert a file.
so simple file comparison is not always sufficient to test a version converter.
For example, none of the following change the meaning of a file in any way:

- A single space after the tag of a structure with no payload.
    
    For example, "`1 CHAN `" and "`1 CHAN`" are equivalent.
    
- Reordering substructures with distinct structure types
    
    For example,

    ````gedcom
    1 CHAN
    2 DATE JUN 2021
    2 NOTE Example
    2 NOTE Order isn't important
    ````
    
    and
    
    ````gedcom
    1 CHAN
    2 NOTE Example
    2 DATE JUN 2021
    2 NOTE Order isn't important
    ````
    
    are equivalent.

- Some datatypes have multiple representations of the same data.
    
    For example, "`GREGORIAN 1930`" and "`1930`" are equivalent in a date payload

- Extensions with URIs don't care about their tags.
    
    For example,
    
    ````gedcom
    1 SCHMA
    2 TAG _X https://example.com/some/structuretype
    0 @X1@ _X the_paylaod
    ````
    
    and
    
    ````gedcom
    1 SCHMA
    2 TAG _STRUCTURETYPE https://example.com/some/structuretype
    0 @X1@ _STRUCTURETYPE the_paylaod
    ````
    
    are equivalent.

- Cross-reference identifiers are arbitrary.

    For example,
    
    ````gedcom
    1 SNOTE @N20@
    0 @N20@ An example note
    ````
    
    and
    
    ````gedcom
    1 SNOTE @80Z_45@
    0 @80Z_45@ An example note
    ````
    
    are equivalent.

There are also some ambiguities in how to perform the conversion.
For example, if a 5.5.1 NOTE_RECORD that is pointed to by exactly one NOTE_STRUCTURE,
if can be correctly converted to 7.0 either by making it a SHARED_NOTE_RECORD or by moving its payload and substructure to the NOTE_STRUCTURE and removing it.

