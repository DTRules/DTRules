# DTRules Website

The marketing and documentation website for DTRules, built with [Astro](https://astro.build/) and React. Includes an interactive poker demo showcasing the decision table rules engine.

## Setup

```sh
npm install
```

## Development

```sh
npm run dev
```

The dev server starts at `http://localhost:4321`.

## Build

```sh
npm run build
```

Production output goes to `./dist/`.

## Preview

```sh
npm run preview
```

## Project Structure

```
website/
├── public/              # Static assets
├── src/
│   ├── components/      # Astro and React components
│   ├── layouts/         # Page layouts (BaseLayout)
│   ├── lib/poker/       # Poker demo engine (TypeScript)
│   ├── pages/           # Routes (index, docs, poker demo)
│   └── styles/          # Global styles
├── astro.config.mjs
├── tailwind.config.mjs
└── package.json
```
