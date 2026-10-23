# entity-comment

A project whose `policy` entity is declared by two EDD files, `v_edd.xml`
and `w_edd.xml`, each with its own entity comment. The comment travels in
the Description cell of the entity's header row on each workbook's EDD
sheet (#1306).

`TestVectorProjectsMatchTheirBuild` checks every project under
`test/vectors` that has an `excel/` directory in both directions:

- `dtrules verify`, Excel into XML: a build from the committed workbooks
  reproduces the committed XML, each file with its own comment;
- an authoring save on a copy, XML into Excel: the refresh re-exports the
  workbooks from the XML, and they must come out byte-identical to the
  committed ones. This is the direction in which a comment was lost, and a
  verify-only check did not see it.

No sample's workbooks carry an entity comment, so this fixture is what
exercises the cell.

The rules are the `otherwise/ok` vectors, brought under the authoring API
(`dtrules table get | put`, `dtrules edd put`), which wrote the workbooks.
