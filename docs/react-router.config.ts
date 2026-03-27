import type { Config } from '@react-router/dev/config';
import { createGetUrl, getSlugs } from 'fumadocs-core/source';
import { glob } from 'tinyglobby';

const getUrl = createGetUrl('/docs');

export default {
  ssr: false,
  future: {
    v8_middleware: true,
  },
  async prerender({ getStaticPaths }) {
    const paths: string[] = [];
    const excluded: string[] = [];

    for (const path of getStaticPaths()) {
      if (!excluded.includes(path)) paths.push(path);
    }

    const entries = await glob('**/*.mdx', { cwd: 'content/docs' });

    for (const entry of entries) {
      const slugs = getSlugs(entry);
      paths.push(getUrl(slugs), `/llms.mdx/docs/${[...slugs, 'index.mdx'].join('/')}`);
    }

    return paths;
  },
} satisfies Config;
