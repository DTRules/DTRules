// @ts-check
import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';
import mdx from '@astrojs/mdx';
import react from '@astrojs/react';
import sitemap from '@astrojs/sitemap';

// https://astro.build/config
export default defineConfig({
  site: 'https://dtrules.com',
  // Astro 7 defaults to compressHTML: 'jsx', which drops the line breaks
  // around inline elements and runs words together; keep Astro 5's behaviour.
  compressHTML: true,
  vite: {
    plugins: [tailwindcss()]
  },
  integrations: [mdx(), react(), sitemap()]
});
