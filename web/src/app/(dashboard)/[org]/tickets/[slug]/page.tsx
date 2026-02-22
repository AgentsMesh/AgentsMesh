"use client";

import { useParams, useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { TicketDetail } from "@/components/tickets";

export default function TicketDetailPage() {
  const params = useParams();
  const router = useRouter();
  const slug = params.slug as string;

  return (
    <div className="p-6">
      {/* Back navigation */}
      <div className="mb-6">
        <Button variant="ghost" onClick={() => router.back()}>
          <svg
            className="w-4 h-4 mr-2"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M10 19l-7-7m0 0l7-7m-7 7h18"
            />
          </svg>
          Back to Tickets
        </Button>
      </div>

      {/* Ticket Detail */}
      <TicketDetail slug={slug} />
    </div>
  );
}
