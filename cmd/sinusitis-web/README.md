# sinusitis-web — embedded interactive web demo (#850/#855)

A single self-contained binary that embeds the SinusitisTherapy rule set
(`//go:embed`) and serves it as an interactive **web interview**. It
demonstrates lazy data collection: the rules execute, and each time they reach
a `collect` field that hasn't been supplied, the browser is asked for it. When
execution finishes, the computed therapy is rendered.

```
go run ./cmd/sinusitis-web        # then open http://localhost:8080/
go run ./cmd/sinusitis-web -addr :9000
```

No `.xlsx`/`.xml` need ship with the binary — the rules are baked in.

## The embedded rules (`rules/xml/`)

`rules/xml/` is a **generated snapshot** of
`sampleprojects/SinusitisTherapy/xml/` with `collect` + question metadata added
to the `patient` input fields (diagnosis, age, lean_body_weight, pcr,
penicillin_allergic). It is the compiled artifact the demo embeds — not a
hand-authored source. The authoritative, Excel-backed project remains
`sampleprojects/SinusitisTherapy`.

To regenerate after the sample changes, copy its `xml/` here and re-apply the
`collect` annotations via the authoring API (`dtrules edd patch`), then drop
the `xml/` tree into `rules/xml/`.
