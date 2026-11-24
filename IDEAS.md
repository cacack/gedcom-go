# Ideas

Unvetted ideas and rough concepts. When ready to implement, create a GitHub issue.

## Encoder Improvements

- **BOM output option**: Optionally write a UTF-8 BOM for Windows compatibility (today a BOM is written only when `PreserveLayout` carries one over from the source)

## API Ideas

- **Fluent builder API**: Build documents with method chaining (`gedcom.NewDocument` and `Document.AddRecord` exist; a chaining builder does not)
