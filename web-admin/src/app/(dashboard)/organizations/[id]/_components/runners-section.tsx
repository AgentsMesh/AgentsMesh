"use client";

import Link from "next/link";
import { Server } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Runner } from "@/lib/api/admin";

interface RunnersSectionProps {
  runners: Runner[];
}

export function RunnersSection({ runners }: RunnersSectionProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Server className="h-5 w-5" />
          Runners ({runners.length})
        </CardTitle>
      </CardHeader>
      <CardContent>
        {runners.length === 0 ? (
          <p className="py-4 text-center text-muted-foreground">No runners found</p>
        ) : (
          <div className="space-y-2">
            {runners.map((runner) => (
              <div
                key={runner.id}
                className="flex items-center justify-between rounded-lg border border-border p-3"
              >
                <div className="flex items-center gap-3">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-secondary">
                    <Server className="h-4 w-4 text-muted-foreground" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{runner.node_id}</span>
                      <Badge variant={runner.status === "online" ? "success" : "secondary"}>
                        {runner.status}
                      </Badge>
                      {!runner.is_enabled && <Badge variant="destructive">Disabled</Badge>}
                    </div>
                    <p className="text-sm text-muted-foreground">
                      {runner.current_pods}/{runner.max_concurrent_pods} pods
                      {runner.runner_version && ` · v${runner.runner_version}`}
                    </p>
                  </div>
                </div>
                <Link href={`/runners?search=${runner.node_id}`}>
                  <Button variant="ghost" size="sm">
                    View
                  </Button>
                </Link>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
