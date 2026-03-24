'use client';

import { Settings } from 'lucide-react';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-[30px] font-bold tracking-tight flex items-center gap-2">
          <Settings className="h-6 w-6" />
          Settings
        </h1>
        <p className="text-base text-muted-foreground">
          Runtime configuration and server settings.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Runtime Configuration</CardTitle>
          <CardDescription>
            Settings will be available once connected to a Kimbap Core instance.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <Settings className="h-10 w-10 text-muted-foreground/40 mb-3" />
            <p className="text-sm text-muted-foreground">
              Configuration management is coming soon.
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              Use <code className="bg-muted px-1.5 py-0.5 rounded font-mono">kimbap config</code>{' '}
              via CLI in the meantime.
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
