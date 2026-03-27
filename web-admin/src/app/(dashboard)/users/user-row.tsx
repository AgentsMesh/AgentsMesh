import { Shield, ShieldOff, UserX, UserCheck, MailCheck, MailX } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { User } from "@/lib/api/admin";
import { formatDate, formatRelativeTime } from "@/lib/utils";

interface UserRowProps {
  user: User;
  onDisable: () => void;
  onEnable: () => void;
  onGrantAdmin: () => void;
  onRevokeAdmin: () => void;
  onVerifyEmail: () => void;
  onUnverifyEmail: () => void;
}

export function UserRow({
  user,
  onDisable,
  onEnable,
  onGrantAdmin,
  onRevokeAdmin,
  onVerifyEmail,
  onUnverifyEmail,
}: UserRowProps) {
  return (
    <div className="flex flex-col gap-3 rounded-lg border border-border p-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-center gap-4">
        {user.avatar_url ? (
          <img src={user.avatar_url} alt={user.username} className="h-10 w-10 rounded-full" />
        ) : (
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/20 text-sm font-medium text-primary">
            {user.username[0].toUpperCase()}
          </div>
        )}
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{user.name || user.username}</span>
            {user.is_system_admin && (
              <Badge variant="default" className="text-xs">
                <Shield className="mr-1 h-3 w-3" />
                Admin
              </Badge>
            )}
            {!user.is_active && (
              <Badge variant="destructive" className="text-xs">Disabled</Badge>
            )}
            {!user.is_email_verified && (
              <Badge variant="outline" className="text-xs">Unverified</Badge>
            )}
          </div>
          <p className="text-sm text-muted-foreground">{user.email}</p>
        </div>
      </div>
      <div className="flex items-center gap-4">
        <div className="hidden text-right text-xs text-muted-foreground sm:block">
          <p>Joined {formatDate(user.created_at)}</p>
          {user.last_login_at && (
            <p>Last login {formatRelativeTime(user.last_login_at)}</p>
          )}
        </div>
        <UserActions
          user={user}
          onDisable={onDisable}
          onEnable={onEnable}
          onGrantAdmin={onGrantAdmin}
          onRevokeAdmin={onRevokeAdmin}
          onVerifyEmail={onVerifyEmail}
          onUnverifyEmail={onUnverifyEmail}
        />
      </div>
    </div>
  );
}

function UserActions({
  user,
  onDisable,
  onEnable,
  onGrantAdmin,
  onRevokeAdmin,
  onVerifyEmail,
  onUnverifyEmail,
}: UserRowProps) {
  return (
    <div className="flex gap-1">
      {user.is_active ? (
        <Button variant="ghost" size="icon" onClick={onDisable} title="Disable user">
          <UserX className="h-4 w-4" />
        </Button>
      ) : (
        <Button variant="ghost" size="icon" onClick={onEnable} title="Enable user">
          <UserCheck className="h-4 w-4" />
        </Button>
      )}
      {user.is_email_verified ? (
        <Button variant="ghost" size="icon" onClick={onUnverifyEmail} title="Unverify email">
          <MailX className="h-4 w-4" />
        </Button>
      ) : (
        <Button variant="ghost" size="icon" onClick={onVerifyEmail} title="Verify email">
          <MailCheck className="h-4 w-4" />
        </Button>
      )}
      {user.is_system_admin ? (
        <Button variant="ghost" size="icon" onClick={onRevokeAdmin} title="Revoke admin">
          <ShieldOff className="h-4 w-4" />
        </Button>
      ) : (
        <Button variant="ghost" size="icon" onClick={onGrantAdmin} title="Grant admin">
          <Shield className="h-4 w-4" />
        </Button>
      )}
    </div>
  );
}
