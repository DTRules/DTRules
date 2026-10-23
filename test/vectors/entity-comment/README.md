# entity-comment

A project whose `policy` entity is declared by two EDD files, `v_edd.xml`
and `w_edd.xml`, each with its own entity comment. The comment travels in
the Description cell of the entity's header row on each workbook's EDD
sheet (#1306), and `dtrules verify` on this project checks that a build
from the workbooks gives each file its own comment back.

`TestVectorProjectsMatchTheirBuild` verifies every project under
`test/vectors` that has an `excel/` directory, the way the samples are
verified. No sample's workbooks carry an entity comment, so this fixture is
what exercises the cell.

The rules are the `otherwise/ok` vectors, brought under the authoring API
(`dtrules table get | put`, `dtrules edd put`), which wrote the workbooks.
