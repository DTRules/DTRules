# DTRules.com New Website Plan

## Overview

Build a modern, responsive website for DTRules that matches the new UI aesthetic (dark theme with blue/purple gradients) and consolidates all documentation in one place.

---

## Tech Stack

| Component | Technology | Reason |
|-----------|------------|--------|
| Framework | **Astro** or **Next.js** | Static site generation, fast, SEO-friendly |
| Styling | **Tailwind CSS** | Matches UI, utility-first |
| Components | **shadcn/ui** | Same components as DTRules UI |
| Hosting | **GitHub Pages** or **Vercel** | Free, automatic HTTPS, easy deployment |
| Domain | **dtrules.com** | Existing domain |

### Recommended: Astro
- Perfect for content-heavy documentation sites
- Can embed React components where needed (like interactive demos)
- Blazing fast static output
- Built-in Markdown/MDX support for documentation

---

## Site Structure

```
dtrules.com/
├── / (Home)
│   ├── Hero section with tagline
│   ├── Key features grid
│   ├── Quick start code example
│   └── Call-to-action buttons
│
├── /features
│   ├── Decision Tables explanation
│   ├── Performance benchmarks
│   ├── Deterministic execution
│   └── DSL support
│
├── /docs
│   ├── /getting-started
│   │   ├── Installation
│   │   ├── Quick Start
│   │   └── CHIP Example walkthrough
│   ├── /concepts
│   │   ├── Decision Tables
│   │   ├── Entity Definitions (EDD)
│   │   ├── The EL DSL
│   │   └── Balanced vs Unbalanced Tables
│   ├── /guides
│   │   ├── Mapping Data
│   │   ├── Deployment
│   │   ├── Configuration
│   │   └── Error Messages
│   └── /api-reference
│       └── Java API docs
│
├── /demo (NEW!)
│   └── Embedded DTRules UI for live testing
│
├── /downloads
│   ├── Latest release
│   ├── Maven/Gradle coordinates
│   └── Release history
│
├── /community
│   ├── GitHub link
│   ├── Google Groups
│   └── Contributing guide
│
└── /about
    ├── Project history
    ├── License (Apache 2.0)
    └── Contact
```

---

## Design System

### Colors (matching UI)
```css
--background: 240 10% 3.9%      /* Near black */
--foreground: 0 0% 98%          /* White text */
--primary: 217 91% 60%          /* Blue */
--accent: 271 91% 65%           /* Purple */
--muted: 240 3.7% 15.9%         /* Dark gray */
--border: 240 3.7% 15.9%        /* Subtle borders */
```

### Gradients
- Hero backgrounds: `from-blue-600 to-purple-600`
- Buttons: `from-blue-600 to-purple-600`
- Cards: Subtle `from-muted/30 to-transparent`

### Typography
- Headings: `font-semibold`
- Body: `text-muted-foreground`
- Code: Monospace with syntax highlighting

---

## Content Migration

### From Old WordPress Site

| Old Page | New Location | Status |
|----------|--------------|--------|
| About DTRules | /features | Update |
| Documentation | /docs/* | Expand |
| Downloads | /downloads | Update with Maven |
| License | /about | Keep |
| Contact | /community | Modernize |
| Blog posts | Archive or remove | Outdated |

### New Content Needed

1. **Getting Started Guide** - Step-by-step tutorial
2. **Interactive Demo** - Embed the React UI
3. **API Reference** - Generated from JavaDoc
4. **Video Tutorials** - Optional future enhancement

---

## DNS Configuration

### What You Need to Provide

To point dtrules.com to the new site, you need to update DNS records in DreamHost. You have two options:

#### Option 1: GitHub Pages (Recommended)
Create/update these DNS records:

```
Type    Name    Value
----    ----    -----
A       @       185.199.108.153
A       @       185.199.109.153
A       @       185.199.110.153
A       @       185.199.111.153
CNAME   www     dtrules.github.io
```

#### Option 2: Vercel
Create/update these DNS records:

```
Type    Name    Value
----    ----    -----
A       @       76.76.21.21
CNAME   www     cname.vercel-dns.com
```

### How to Access DreamHost DNS

1. Log into DreamHost Panel: https://panel.dreamhost.com
2. Go to: Domains > Manage Domains
3. Click "DNS" under dtrules.com
4. Add/modify the records above

**You don't need to give me credentials** - just make these changes yourself, or share screenshots of the current DNS settings and I can tell you exactly what to change.

---

## Implementation Phases

### Phase 1: Foundation (Week 1)
- [ ] Set up Astro project with Tailwind
- [ ] Implement design system (colors, typography)
- [ ] Create base layout and navigation
- [ ] Build home page

### Phase 2: Content (Week 2)
- [ ] Migrate documentation from WordPress
- [ ] Create Getting Started guide
- [ ] Build features page
- [ ] Add downloads page with GitHub releases

### Phase 3: Interactive Features (Week 3)
- [ ] Embed DTRules UI demo
- [ ] Add search functionality
- [ ] Implement dark/light mode toggle (optional)

### Phase 4: Launch (Week 4)
- [ ] Configure DNS
- [ ] Set up GitHub Actions for deployment
- [ ] Test SSL certificate
- [ ] Go live!

---

## File Structure for New Site

```
dtrules-website/
├── src/
│   ├── components/
│   │   ├── Header.astro
│   │   ├── Footer.astro
│   │   ├── Hero.astro
│   │   ├── FeatureCard.astro
│   │   └── CodeBlock.astro
│   ├── layouts/
│   │   ├── BaseLayout.astro
│   │   └── DocsLayout.astro
│   ├── pages/
│   │   ├── index.astro
│   │   ├── features.astro
│   │   ├── downloads.astro
│   │   ├── about.astro
│   │   └── docs/
│   │       └── [...slug].astro
│   ├── content/
│   │   └── docs/
│   │       ├── getting-started/
│   │       ├── concepts/
│   │       └── guides/
│   └── styles/
│       └── global.css
├── public/
│   ├── favicon.ico
│   └── images/
├── astro.config.mjs
├── tailwind.config.js
└── package.json
```

---

## Questions to Decide

1. **Hosting preference**: GitHub Pages (simpler) or Vercel (more features)?
2. **Demo embedding**: Full UI or simplified demo version?
3. **Search**: Algolia (external) or built-in search?
4. **Analytics**: Add Google Analytics or privacy-focused alternative?

---

## Next Steps

1. You update DNS records (I'll provide exact values once you choose hosting)
2. I create the website project in `/home/paul/DTRules/website/`
3. We iterate on design and content
4. Deploy and go live

Ready to start when you are!
