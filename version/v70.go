package version

// GEDCOM 7.0 tag list
// GEDCOM 7.0 has significant changes from 5.x versions

var gedcom70Tags = []string{
	// Standard record tags
	"HEAD", "TRLR", "INDI", "FAM", "OBJE", "SNOTE", "SOUR", "REPO", "SUBM",

	// Individual/Family event tags
	"ADOP", "ANUL", "BAPM", "BARM", "BASM", "BIRT", "BLES", "BURI",
	"CENS", "CHR", "CHRA", "CONF", "CREM", "DEAT", "DIV", "DIVF",
	"EMIG", "ENGA", "EVEN", "FCOM", "GRAD", "IMMI", "MARB", "MARC",
	"MARR", "MARL", "NATU", "ORDN", "PROB", "RETI", "WILL",

	// Attribute tags
	"CAST", "DSCR", "EDUC", "FACT", "IDNO", "NATI", "NCHI", "NMR",
	"OCCU", "PROP", "RELI", "RESI", "SSN", "TITL",

	// Structure tags
	"ABBR", "ADDR", "ADR1", "ADR2", "ADR3", "AGE", "AGNC", "ALIA",
	"ASSO", "AUTH", "CALN", "CAUS", "CHAN", "CHIL", "CITY", "CONT",
	"COPR", "CORP", "CTRY", "DATA", "DATE", "EMAIL", "EXID", "FAX",
	"FILE", "FORM", "GEDC", "GIVN", "HUSB", "LANG", "LATI", "LONG",
	"MAP", "MEDI", "NAME", "NICK", "NPFX", "NSFX", "NOTE", "PAGE",
	"PEDI", "PHON", "PHRASE", "PLAC", "POST", "PUBL", "QUAY", "REFN",
	"RELA", "REPO", "RESN", "ROLE", "SEX", "SLAT", "SLON", "SLOC",
	"SNOTE", "SOUR", "SPFX", "STAE", "STAT", "SUBM", "SURN", "TAG",
	"TEMP", "TEXT", "TIME", "TOP", "TRAN", "TYPE", "UID", "VERS",
	"WIFE", "WWW",

	// GEDCOM 7.0 specific tags
	"CREA", "CROP", "EXID", "MIME", "NO", "PEDI", "PHRASE", "RESN",
	"SCHMA", "SDATE", "SHARED_NOTE", "SNOTE", "TAG", "TOP", "TRAN",
	"TYPE", "UID",
}
