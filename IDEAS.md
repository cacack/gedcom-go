# Ideas

Unvetted ideas and rough concepts. When ready to implement, create a GitHub issue.

## Encoder Improvements

- **CONC/CONT line splitting**: Automatically split long lines per GEDCOM spec (255 char limit)
- **BOM output option**: Optionally write UTF-8 BOM for Windows compatibility
- **Entity-aware encoding**: Encode structured types (Individual, Family) directly, not just raw records

## Parser/Decoder Improvements

- **Association source citations**: Parse SOUR subordinates under ASSO tags
- **Loose parsing mode**: Accept common non-standard variations found in the wild

## API Ideas

- **Fluent builder API**: Build documents with method chaining
- **JSON struct tags**: Add json tags to types for easy serialization
