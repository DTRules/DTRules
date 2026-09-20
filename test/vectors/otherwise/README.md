# Otherwise-column vectors (#1215)

`*` does **not** mean "don't care". It is allowed only in the **last** column, and only when that column has
no `Y`/`N` entries. It means **otherwise**: it executes only if no other column executes, in **all** table
types. There is no "always" column — to always execute an action, put an `X` in every column.

`python3 test/vectors/otherwise/run.py [engine|authoring|docs|all]` builds `./cmd/dtrules` and checks that.
These vectors were written before the fix and are not the implementer's to edit; if one looks wrong, say so.

Assumption to confirm: in XML an empty cell is `-`, so an otherwise column is one `*` plus `-` cells
(`V9`, `V10`), or `*` in every cell (`V1`). Both are accepted as "a column with no Y/N entries".
Fixtures carry hand-written postfix only because the authoring API could not express `*` when they were made.
