import type { Route } from './+types/home';
import { HomeLayout } from 'fumadocs-ui/layouts/home';
import { Link } from 'react-router';
import { baseOptions } from '@/lib/layout.shared';

export function meta({}: Route.MetaArgs) {
  return [
    { title: 'fwdx Docs' },
    { name: 'description', content: 'Documentation for the fwdx self-hosted tunnel platform.' },
  ];
}

export default function Home() {
  return (
    <HomeLayout {...baseOptions()}>
      <div className="p-4 flex flex-col items-center justify-center text-center flex-1">
        <h1 className="text-xl font-bold mb-2">fwdx Documentation</h1>
        <p className="text-fd-muted-foreground mb-4">
          Self-hosted HTTP tunneling, deployment, CLI usage, and architecture.
        </p>
        <Link
          className="text-sm bg-fd-primary text-fd-primary-foreground rounded-full font-medium px-4 py-2.5"
          to="/docs"
        >
          Open Documentation
        </Link>
      </div>
    </HomeLayout>
  );
}
