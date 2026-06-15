# Upstash Redis Local — Documentation

Mintlify-powered documentation site for [Upstash Redis Local](https://github.com/aine1100/Upstash-Redis-Local-server).

Built with [Mintlify](https://www.mintlify.com/docs) — AI-native docs with beautiful defaults, search, and dark mode.

## Preview locally

Requires Node.js v20.17.0+:

```bash
npm i -g mint
cd docs
mint dev
```

Open **http://localhost:3000**

## Deploy to Mintlify

1. Go to [mintlify.com/start](https://mintlify.com/start)
2. Connect your GitHub repository
3. Set the docs directory to `docs/`
4. Deploy — your site will be live at `https://your-project.mintlify.app`

Or connect the `docs/` folder as a subdirectory of your main repo.

## Structure

```
docs/
├── docs.json           # Site config & navigation
├── index.mdx           # Introduction
├── quickstart.mdx
├── installation/
├── guides/
├── api-reference/
├── testing/
├── troubleshooting.mdx
└── contributing.mdx
```

## Edit a page

Each page is an MDX file with frontmatter:

```mdx
---
title: Page Title
description: Short description for SEO and search.
---

Your content here. Use Mintlify components:
<Note>, <Tip>, <Warning>, <Steps>, <CardGroup>, etc.
```

See [Mintlify components](https://www.mintlify.com/docs/components/accordions).
