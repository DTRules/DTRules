# CorporateTax reference sources

Official state corporate-tax forms, instructions and published regulations, for
authoring and checking the rules in `../xml/states/`.

CorporateTax is the only sample project with no Excel authoring source — its
XML was hand-written rather than derived from anything. That is how it ended up
with 130 unescaped ampersands and `<field>` elements split down the middle, and
it is why there was no reference material to check a single tax rate against.
This directory is that reference.

## What is here

- **`sources.tsv`** — jurisdiction, document, URL. The list of what we intend to
  have. Edit this to add a document.
- **`manifest.tsv`** — what was actually retrieved: file, size, sha256, date,
  URL. This is the provenance record. A rate or rule traced to a form can be
  re-checked against the exact bytes that were read.
- **`fetch.sh`** — downloads everything in `sources.tsv`, verifies each response
  really is a PDF (a state site that has retired a form answers `200` with an
  HTML page), and rewrites the manifest.
- **`forms/<STATE>/`** — the PDFs. **Gitignored.** They are ~86 MB of
  republished material that is reissued every filing season; the repo keeps the
  manifest instead. Run `./fetch.sh` to populate.

```bash
./fetch.sh            # everything
./fetch.sh CA NY TX   # named jurisdictions only
```

## Coverage

**43 of 51 jurisdictions, 76 documents, 85.9 MB** as of 2026-08-02.

Where a state publishes its corporate-tax regulations or an audit manual as a
PDF, that is included too — Pennsylvania's Corporate Net Income Tax Audit
Manual, Tennessee's Franchise & Excise Tax Manual, Maine's Rule 104, Delaware's
statutory-provision review, Wisconsin's DOR reference report. Most states
publish their administrative code as HTML only; see "Regulations" below.

### Not retrieved

| Jurisdiction | Why |
|---|---|
| MA | mass.gov returns `403` to scripted requests, with or without browser headers |
| NH | revenue.nh.gov returns `403` the same way — including its Rev 300 BPT rules |
| MS | dor.ms.gov fails TLS chain verification. Not worked around: disabling certificate checks to fetch a document we intend to treat as authoritative is the wrong trade |
| AK | Form 6000 is served through a document-viewer endpoint, not a direct PDF |
| NM | no official CIT-1 PDF found on tax.newmexico.gov; only third-party mirrors, which are not authoritative for tax rules |
| OH | commercial activity tax is e-file only — the state publishes no annual-return PDF |
| SD, WY | **no corporate income tax and no equivalent gross-receipts tax.** Nothing to download; the absence is the fact |

The URLs for MA, NH and MS are in `sources.tsv` and are correct — they need a
browser, not a fix to the script.

Three more jurisdictions have no corporate income tax as such and are
represented by their nearest equivalent: **NV** (Commerce Tax), **TX**
(Franchise Tax), **WA** (B&O within the Combined Excise Tax Return).

## Regulations

"All state regs" is not a set of files. State administrative codes are HTML
corpora — California Code of Regulations Title 18, New York 20 NYCRR, and so on
— running to tens of thousands of pages, most without a canonical PDF edition.
Downloading them wholesale would produce something unusable.

What is captured instead:

1. **Instructions booklets**, which is where the operative computational rules
   actually live and what a rule author works from day to day.
2. **Regulations and audit manuals that a state does publish as a PDF** (PA, TN,
   ME, NH once reachable).
3. **Statutory citations**, which the decision tables already carry inline —
   `IRC § 1231`, `PR Act 60-2019 § 20.02`, `Rule 12C-1.051`. Those are the
   authoritative pointers; they resolve against the state code.

If a specific state's regulations are needed in full, the right move is to fetch
that one state's code chapter deliberately, not to mirror fifty states.

## Provenance and staleness

Every row in `manifest.tsv` carries a retrieval date and a sha256. Forms are
reissued annually, and several already carry a later revision than the tax year
they cover — Alabama's 2025 Form 20C was posted in January 2026, Idaho's Form 41
in March 2026. Treat the manifest date, not the form year, as the age of the
evidence, and re-run `fetch.sh` before relying on a rate for a new filing
season.

These are US government works, republished here unmodified for reference.
