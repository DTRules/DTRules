# Field-constraint enforcement vectors (#1209 Part B)

Part A (#1214) lets a field *declare* `allowed_values`, `max_length`, `max_words`. These vectors define Part B:
every path that writes a field from **outside** the rules refuses a violating value — non-zero exit, an error
naming `entity.field`, the offending value and the allowed set (or the limit and the actual size), and **no
result printed**. Matching is case-insensitive. An absent field takes its default without error. A rule's own
`set` is not refused; `dtrules review` flags a literal outside the set as an advisory.

`python3 test/vectors/constraints/run.py` builds `./cmd/dtrules` and checks that against a patched copy of
SinusitisTherapy. Written before the fix; not the implementer's to edit — if one looks wrong, say so.

Not expressible from the CLI, so required as Go tests instead: the `collect` resolver (issue vector 5 — an
`Asker` returning a bad value makes `Put` fail, aborts the run, and does not leave the field marked collected),
the web interview, and the API server.
