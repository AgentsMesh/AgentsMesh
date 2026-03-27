import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export function SeatsPanel({
  seatUsage,
  newSeatCount,
  onNewSeatCountChange,
  onSetSeats,
  seatsPending,
}: {
  seatUsage?: { total_seats: number; used_seats: number; available_seats: number };
  newSeatCount: string;
  onNewSeatCountChange: (v: string) => void;
  onSetSeats: (count: number) => void;
  seatsPending: boolean;
}) {
  return (
    <div className="space-y-3 rounded-lg border border-border p-4">
      <h3 className="text-sm font-semibold text-muted-foreground">Seats</h3>
      {seatUsage ? (
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span>{seatUsage.used_seats}/{seatUsage.total_seats} used</span>
            <span className="text-muted-foreground">{seatUsage.available_seats} available</span>
          </div>
          <div className="h-2 rounded-full bg-muted">
            <div
              className="h-2 rounded-full bg-primary transition-all"
              style={{ width: `${seatUsage.total_seats > 0 ? Math.min((seatUsage.used_seats / seatUsage.total_seats) * 100, 100) : 0}%` }}
            />
          </div>
          <div className="flex items-center gap-2">
            <Input
              type="number"
              min={1}
              placeholder="New count"
              value={newSeatCount}
              onChange={(e) => onNewSeatCountChange(e.target.value)}
              className="h-8 w-24 text-sm"
            />
            <Button
              variant="outline"
              size="sm"
              className="h-8 text-xs"
              disabled={!newSeatCount || seatsPending}
              onClick={() => {
                const count = parseInt(newSeatCount);
                if (count > 0) onSetSeats(count);
              }}
            >
              Set Seats
            </Button>
          </div>
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">No seat data</p>
      )}
    </div>
  );
}
