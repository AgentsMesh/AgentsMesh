"use client";

import { useState, useEffect, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuthStore } from "@/stores/auth";
import { organizationApi, agentApi, billingApi, BillingOverview, SubscriptionPlan, gitProviderApi, sshKeyApi, SSHKeyData } from "@/lib/api/client";
import { useRunnerStore, Runner, RegistrationToken, getRunnerStatusInfo } from "@/stores/runner";
import { NotificationSettings } from "@/components/settings";

export default function SettingsPage() {
  const searchParams = useSearchParams();
  const activeTab = searchParams.get("tab") || "general";
  const { currentOrg } = useAuthStore();

  // Tab content mapping
  const renderContent = () => {
    switch (activeTab) {
      case "general":
        return <GeneralSettings org={currentOrg} />;
      case "members":
        return <MembersSettings />;
      case "agents":
        return <AgentsSettings />;
      case "runners":
        return <RunnersSettings />;
      case "git-providers":
        return <GitProvidersSettings />;
      case "notifications":
        return <NotificationsSettings />;
      case "billing":
        return <BillingSettings />;
      default:
        return <GeneralSettings org={currentOrg} />;
    }
  };

  return (
    <div className="h-full overflow-auto p-6">
      {/* Content - navigation controlled by IDE Sidebar */}
      <div className="max-w-4xl">
        {renderContent()}
      </div>
    </div>
  );
}

function GeneralSettings({ org }: { org: { name: string; slug: string } | null }) {
  const [name, setName] = useState(org?.name || "");
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    try {
      await organizationApi.update(org!.slug, { name });
    } catch (error) {
      console.error("Failed to save:", error);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">Organization Details</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">
              Organization Name
            </label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My Organization"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">
              Organization Slug
            </label>
            <Input value={org?.slug || ""} disabled />
            <p className="text-xs text-muted-foreground mt-1">
              The slug cannot be changed after creation
            </p>
          </div>
        </div>
        <div className="mt-6">
          <Button onClick={handleSave} disabled={saving}>
            {saving ? "Saving..." : "Save Changes"}
          </Button>
        </div>
      </div>

      <div className="border border-destructive rounded-lg p-6">
        <h2 className="text-lg font-semibold text-destructive mb-4">
          Danger Zone
        </h2>
        <p className="text-sm text-muted-foreground mb-4">
          Once you delete an organization, there is no going back. Please be
          certain.
        </p>
        <Button variant="destructive">Delete Organization</Button>
      </div>
    </div>
  );
}

function MembersSettings() {
  const { currentOrg, user } = useAuthStore();
  const [members, setMembers] = useState<Array<{
    id: number;
    user_id: number;
    role: string;
    joined_at: string;
    user?: { id: number; email: string; username: string; name?: string };
  }>>([]);
  const [loading, setLoading] = useState(true);
  const [showInviteDialog, setShowInviteDialog] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("member");
  const [inviting, setInviting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadMembers = useCallback(async () => {
    if (!currentOrg) return;
    try {
      setLoading(true);
      const response = await organizationApi.listMembers(currentOrg.slug);
      setMembers(response.members || []);
    } catch (err) {
      console.error("Failed to load members:", err);
      setError("Failed to load members");
    } finally {
      setLoading(false);
    }
  }, [currentOrg]);

  useEffect(() => {
    loadMembers();
  }, [loadMembers]);

  const handleInvite = async () => {
    if (!currentOrg || !inviteEmail) return;
    setInviting(true);
    setError(null);
    try {
      await organizationApi.inviteMember(currentOrg.slug, inviteEmail, inviteRole);
      setShowInviteDialog(false);
      setInviteEmail("");
      setInviteRole("member");
      await loadMembers();
    } catch (err) {
      console.error("Failed to invite member:", err);
      setError("Failed to invite member. Please check the email and try again.");
    } finally {
      setInviting(false);
    }
  };

  const handleRemove = async (userId: number) => {
    if (!currentOrg) return;
    if (!confirm("Are you sure you want to remove this member?")) return;
    try {
      await organizationApi.removeMember(currentOrg.slug, userId);
      await loadMembers();
    } catch (err) {
      console.error("Failed to remove member:", err);
      setError("Failed to remove member");
    }
  };

  const handleRoleChange = async (userId: number, newRole: string) => {
    if (!currentOrg) return;
    try {
      await organizationApi.updateMemberRole(currentOrg.slug, userId, newRole);
      await loadMembers();
    } catch (err) {
      console.error("Failed to update role:", err);
      setError("Failed to update member role");
    }
  };

  const getRoleBadgeColor = (role: string) => {
    switch (role) {
      case "owner": return "bg-purple-100 text-purple-800";
      case "admin": return "bg-blue-100 text-blue-800";
      default: return "bg-gray-100 text-gray-800";
    }
  };

  return (
    <div className="border border-border rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">Members</h2>
          <p className="text-sm text-muted-foreground">
            Manage who has access to this organization
          </p>
        </div>
        <Button onClick={() => setShowInviteDialog(true)}>Invite Member</Button>
      </div>

      {error && (
        <div className="bg-destructive/10 border border-destructive text-destructive px-4 py-3 rounded-lg mb-4">
          {error}
          <button onClick={() => setError(null)} className="ml-4 underline text-sm">
            Dismiss
          </button>
        </div>
      )}

      {loading ? (
        <div className="text-center py-8 text-muted-foreground">Loading members...</div>
      ) : members.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          No members found. Invite someone to get started.
        </div>
      ) : (
        <div className="space-y-3">
          {members.map((member) => (
            <div
              key={member.id}
              className="flex items-center justify-between p-4 border border-border rounded-lg"
            >
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center text-sm font-medium">
                  {member.user?.name?.[0] || member.user?.username?.[0] || "?"}
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium">
                      {member.user?.name || member.user?.username || "Unknown"}
                    </span>
                    <span className={`text-xs px-2 py-0.5 rounded-full ${getRoleBadgeColor(member.role)}`}>
                      {member.role}
                    </span>
                    {member.user_id === user?.id && (
                      <span className="text-xs text-muted-foreground">(You)</span>
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground">{member.user?.email}</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                {member.role !== "owner" && member.user_id !== user?.id && (
                  <>
                    <select
                      value={member.role}
                      onChange={(e) => handleRoleChange(member.user_id, e.target.value)}
                      className="text-sm border border-border rounded px-2 py-1 bg-background"
                    >
                      <option value="member">Member</option>
                      <option value="admin">Admin</option>
                    </select>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => handleRemove(member.user_id)}
                    >
                      Remove
                    </Button>
                  </>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Invite Dialog */}
      {showInviteDialog && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background border border-border rounded-lg p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold mb-4">Invite Member</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">Email Address</label>
                <Input
                  type="email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  placeholder="colleague@example.com"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">Role</label>
                <select
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value)}
                  className="w-full border border-border rounded px-3 py-2 bg-background"
                >
                  <option value="member">Member</option>
                  <option value="admin">Admin</option>
                </select>
              </div>
            </div>
            <div className="flex gap-3 mt-6">
              <Button
                variant="outline"
                className="flex-1"
                onClick={() => {
                  setShowInviteDialog(false);
                  setInviteEmail("");
                  setInviteRole("member");
                }}
              >
                Cancel
              </Button>
              <Button
                className="flex-1"
                onClick={handleInvite}
                disabled={inviting || !inviteEmail}
              >
                {inviting ? "Inviting..." : "Send Invite"}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function AgentsSettings() {
  const [agentTypes, setAgentTypes] = useState<
    Array<{ id: number; slug: string; name: string; description?: string }>
  >([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Credentials state
  const [anthropicKey, setAnthropicKey] = useState("");
  const [openaiKey, setOpenaiKey] = useState("");
  const [googleKey, setGoogleKey] = useState("");
  const [savingCredentials, setSavingCredentials] = useState(false);

  useEffect(() => {
    loadAgentTypes();
  }, []);

  const loadAgentTypes = async () => {
    try {
      const response = await agentApi.listTypes();
      setAgentTypes(response.agent_types || []);
    } catch (error) {
      console.error("Failed to load agent types:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleSaveCredentials = async () => {
    setSavingCredentials(true);
    setError(null);
    setSuccess(null);
    try {
      // Save credentials for each provider that has a value
      const promises = [];
      if (anthropicKey) {
        promises.push(
          agentApi.updateCredentials("claude", { api_key: anthropicKey })
        );
      }
      if (openaiKey) {
        promises.push(
          agentApi.updateCredentials("openai", { api_key: openaiKey })
        );
      }
      if (googleKey) {
        promises.push(
          agentApi.updateCredentials("gemini", { api_key: googleKey })
        );
      }

      if (promises.length === 0) {
        setError("Please enter at least one API key to save");
        return;
      }

      await Promise.all(promises);
      setSuccess("Credentials saved successfully");
      // Clear the inputs after saving
      setAnthropicKey("");
      setOpenaiKey("");
      setGoogleKey("");
    } catch (err) {
      console.error("Failed to save credentials:", err);
      setError("Failed to save credentials. Please try again.");
    } finally {
      setSavingCredentials(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">Agent Configuration</h2>
        <p className="text-sm text-muted-foreground mb-4">
          Enable and configure AI agents for your organization
        </p>

        {loading ? (
          <div className="text-center py-4">Loading...</div>
        ) : (
          <div className="space-y-4">
            {agentTypes.map((agent) => (
              <div
                key={agent.id}
                className="flex items-center justify-between p-4 border border-border rounded-lg"
              >
                <div>
                  <h3 className="font-medium">{agent.name}</h3>
                  <p className="text-sm text-muted-foreground">
                    {agent.description || `Configure ${agent.name} settings`}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Button variant="outline" size="sm">
                    Configure
                  </Button>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input type="checkbox" className="sr-only peer" />
                    <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
                  </label>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">Your Credentials</h2>
        <p className="text-sm text-muted-foreground mb-4">
          Set your personal API keys for AI providers. Keys are encrypted before storage.
        </p>

        {error && (
          <div className="bg-destructive/10 border border-destructive text-destructive px-4 py-3 rounded-lg mb-4">
            {error}
            <button onClick={() => setError(null)} className="ml-4 underline text-sm">
              Dismiss
            </button>
          </div>
        )}

        {success && (
          <div className="bg-green-50 border border-green-500 text-green-700 px-4 py-3 rounded-lg mb-4">
            {success}
            <button onClick={() => setSuccess(null)} className="ml-4 underline text-sm">
              Dismiss
            </button>
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">
              Anthropic API Key (Claude)
            </label>
            <Input
              type="password"
              placeholder="sk-ant-..."
              value={anthropicKey}
              onChange={(e) => setAnthropicKey(e.target.value)}
            />
            <p className="text-xs text-muted-foreground mt-1">
              Get your key from{" "}
              <a
                href="https://console.anthropic.com/settings/keys"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                console.anthropic.com
              </a>
            </p>
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">
              OpenAI API Key
            </label>
            <Input
              type="password"
              placeholder="sk-..."
              value={openaiKey}
              onChange={(e) => setOpenaiKey(e.target.value)}
            />
            <p className="text-xs text-muted-foreground mt-1">
              Get your key from{" "}
              <a
                href="https://platform.openai.com/api-keys"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                platform.openai.com
              </a>
            </p>
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">
              Google AI API Key (Gemini)
            </label>
            <Input
              type="password"
              placeholder="AIza..."
              value={googleKey}
              onChange={(e) => setGoogleKey(e.target.value)}
            />
            <p className="text-xs text-muted-foreground mt-1">
              Get your key from{" "}
              <a
                href="https://aistudio.google.com/app/apikey"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                aistudio.google.com
              </a>
            </p>
          </div>
        </div>
        <div className="mt-4">
          <Button
            onClick={handleSaveCredentials}
            disabled={savingCredentials || (!anthropicKey && !openaiKey && !googleKey)}
          >
            {savingCredentials ? "Saving..." : "Save Credentials"}
          </Button>
        </div>
      </div>
    </div>
  );
}

function GitProvidersSettings() {
  const [providers, setProviders] = useState<Array<{
    id: number;
    provider_type: string;
    name: string;
    base_url: string;
    ssh_key_id?: number;
    is_default: boolean;
    is_active: boolean;
  }>>([]);
  const [sshKeys, setSSHKeys] = useState<SSHKeyData[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [editingProvider, setEditingProvider] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // SSH Key management
  const [showSSHKeyDialog, setShowSSHKeyDialog] = useState(false);
  const [newSSHKeyName, setNewSSHKeyName] = useState("");
  const [newSSHKeyPrivate, setNewSSHKeyPrivate] = useState("");
  const [createdSSHKey, setCreatedSSHKey] = useState<SSHKeyData | null>(null);
  const [savingSSHKey, setSavingSSHKey] = useState(false);

  // Form states
  const [formType, setFormType] = useState("github");
  const [formName, setFormName] = useState("");
  const [formBaseUrl, setFormBaseUrl] = useState("");
  const [formClientId, setFormClientId] = useState("");
  const [formClientSecret, setFormClientSecret] = useState("");
  const [formBotToken, setFormBotToken] = useState("");
  const [formSSHKeyId, setFormSSHKeyId] = useState<number | null>(null);
  const [formIsDefault, setFormIsDefault] = useState(false);
  const [saving, setSaving] = useState(false);

  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      const [providersRes, sshKeysRes] = await Promise.all([
        gitProviderApi.list(),
        sshKeyApi.list(),
      ]);
      setProviders(providersRes.git_providers || []);
      setSSHKeys(sshKeysRes.ssh_keys || []);
    } catch (err) {
      console.error("Failed to load data:", err);
      setError("Failed to load git providers");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const getDefaultBaseUrl = (type: string) => {
    switch (type) {
      case "github": return "https://github.com";
      case "gitlab": return "https://gitlab.com";
      case "gitee": return "https://gitee.com";
      case "ssh": return "";
      default: return "";
    }
  };

  const resetForm = () => {
    setFormType("github");
    setFormName("");
    setFormBaseUrl("");
    setFormClientId("");
    setFormClientSecret("");
    setFormBotToken("");
    setFormSSHKeyId(null);
    setFormIsDefault(false);
    setEditingProvider(null);
  };

  const handleAddProvider = async () => {
    setSaving(true);
    setError(null);
    try {
      await gitProviderApi.create({
        provider_type: formType,
        name: formName || `${formType.charAt(0).toUpperCase() + formType.slice(1)}`,
        base_url: formBaseUrl || getDefaultBaseUrl(formType),
        client_id: formClientId || undefined,
        client_secret: formClientSecret || undefined,
        bot_token: formBotToken || undefined,
        ssh_key_id: formType === "ssh" && formSSHKeyId ? formSSHKeyId : undefined,
        is_default: formIsDefault,
      });
      setShowAddDialog(false);
      resetForm();
      await loadData();
    } catch (err) {
      console.error("Failed to add provider:", err);
      setError("Failed to add provider");
    } finally {
      setSaving(false);
    }
  };

  const handleUpdateProvider = async (id: number) => {
    setSaving(true);
    setError(null);
    try {
      await gitProviderApi.update(id, {
        name: formName || undefined,
        base_url: formBaseUrl || undefined,
        client_id: formClientId || undefined,
        client_secret: formClientSecret || undefined,
        bot_token: formBotToken || undefined,
        ssh_key_id: formType === "ssh" && formSSHKeyId ? formSSHKeyId : undefined,
        is_default: formIsDefault,
      });
      setEditingProvider(null);
      resetForm();
      await loadData();
    } catch (err) {
      console.error("Failed to update provider:", err);
      setError("Failed to update provider");
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteProvider = async (id: number) => {
    if (!confirm("Are you sure you want to delete this provider?")) return;
    try {
      await gitProviderApi.delete(id);
      await loadData();
    } catch (err) {
      console.error("Failed to delete provider:", err);
      setError("Failed to delete provider");
    }
  };

  const handleToggleActive = async (id: number, isActive: boolean) => {
    try {
      await gitProviderApi.update(id, { is_active: !isActive });
      await loadData();
    } catch (err) {
      console.error("Failed to toggle provider:", err);
      setError("Failed to toggle provider status");
    }
  };

  const openEditDialog = (provider: typeof providers[0]) => {
    setFormType(provider.provider_type);
    setFormName(provider.name);
    setFormBaseUrl(provider.base_url);
    setFormSSHKeyId(provider.ssh_key_id || null);
    setFormIsDefault(provider.is_default);
    setEditingProvider(provider.id);
  };

  const handleCreateSSHKey = async () => {
    if (!newSSHKeyName) {
      setError("SSH key name is required");
      return;
    }
    setSavingSSHKey(true);
    setError(null);
    try {
      const res = await sshKeyApi.create({
        name: newSSHKeyName,
        private_key: newSSHKeyPrivate || undefined, // If empty, generate new key pair
      });
      setCreatedSSHKey(res.ssh_key);
      setSuccess("SSH key created successfully");
      await loadData();
    } catch (err) {
      console.error("Failed to create SSH key:", err);
      setError("Failed to create SSH key");
    } finally {
      setSavingSSHKey(false);
    }
  };

  const handleDeleteSSHKey = async (id: number) => {
    if (!confirm("Are you sure you want to delete this SSH key?")) return;
    try {
      await sshKeyApi.delete(id);
      await loadData();
      setSuccess("SSH key deleted");
    } catch (err) {
      console.error("Failed to delete SSH key:", err);
      setError("Failed to delete SSH key");
    }
  };

  const resetSSHKeyDialog = () => {
    setShowSSHKeyDialog(false);
    setNewSSHKeyName("");
    setNewSSHKeyPrivate("");
    setCreatedSSHKey(null);
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const getProviderIcon = (type: string) => {
    switch (type) {
      case "github": return "GH";
      case "gitlab": return "GL";
      case "gitee": return "GE";
      case "ssh": return "🔑";
      default: return "?";
    }
  };

  const isSSHProvider = formType === "ssh";

  return (
    <div className="space-y-6">
      {/* Git Providers */}
      <div className="border border-border rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold">Git Providers</h2>
            <p className="text-sm text-muted-foreground">
              Configure Git providers for repository integration
            </p>
          </div>
          <Button onClick={() => setShowAddDialog(true)}>Add Provider</Button>
        </div>

        {error && (
          <div className="bg-destructive/10 border border-destructive text-destructive px-4 py-3 rounded-lg mb-4">
            {error}
            <button onClick={() => setError(null)} className="ml-4 underline text-sm">
              Dismiss
            </button>
          </div>
        )}

        {success && (
          <div className="bg-green-50 border border-green-500 text-green-700 px-4 py-3 rounded-lg mb-4">
            {success}
            <button onClick={() => setSuccess(null)} className="ml-4 underline text-sm">
              Dismiss
            </button>
          </div>
        )}

        {loading ? (
          <div className="text-center py-8 text-muted-foreground">Loading providers...</div>
        ) : providers.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            No Git providers configured. Add one to get started.
          </div>
        ) : (
          <div className="space-y-4">
            {providers.map((provider) => (
              <div
                key={provider.id}
                className={`flex items-center justify-between p-4 border border-border rounded-lg ${
                  !provider.is_active ? "opacity-60" : ""
                }`}
              >
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-lg bg-muted flex items-center justify-center font-medium">
                    {getProviderIcon(provider.provider_type)}
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="font-medium">{provider.name}</h3>
                      <span className="text-xs bg-muted px-2 py-0.5 rounded">
                        {provider.provider_type.toUpperCase()}
                      </span>
                      {provider.is_default && (
                        <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded">
                          Default
                        </span>
                      )}
                      {!provider.is_active && (
                        <span className="text-xs bg-yellow-100 text-yellow-800 px-2 py-0.5 rounded">
                          Disabled
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground">
                      {provider.base_url || (provider.provider_type === "ssh" ? "SSH Authentication" : "")}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Button variant="outline" size="sm" onClick={() => openEditDialog(provider)}>
                    Configure
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleToggleActive(provider.id, provider.is_active)}
                  >
                    {provider.is_active ? "Disable" : "Enable"}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-destructive hover:text-destructive"
                    onClick={() => handleDeleteProvider(provider.id)}
                  >
                    Delete
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Add/Edit Dialog */}
        {(showAddDialog || editingProvider !== null) && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-background border border-border rounded-lg p-6 w-full max-w-md max-h-[90vh] overflow-y-auto">
              <h3 className="text-lg font-semibold mb-4">
                {editingProvider ? "Configure Provider" : "Add Git Provider"}
              </h3>
              <div className="space-y-4">
                {!editingProvider && (
                  <div>
                    <label className="block text-sm font-medium mb-2">Provider Type</label>
                    <select
                      value={formType}
                      onChange={(e) => {
                        setFormType(e.target.value);
                        setFormBaseUrl(getDefaultBaseUrl(e.target.value));
                      }}
                      className="w-full border border-border rounded px-3 py-2 bg-background"
                    >
                      <option value="github">GitHub</option>
                      <option value="gitlab">GitLab</option>
                      <option value="gitee">Gitee</option>
                      <option value="ssh">SSH (Generic)</option>
                    </select>
                    {isSSHProvider && (
                      <p className="text-xs text-muted-foreground mt-1">
                        SSH provider uses SSH keys for authentication. Suitable for self-hosted Git servers.
                      </p>
                    )}
                  </div>
                )}
                <div>
                  <label className="block text-sm font-medium mb-2">Name</label>
                  <Input
                    value={formName}
                    onChange={(e) => setFormName(e.target.value)}
                    placeholder={isSSHProvider ? "e.g., My Git Server" : "e.g., My GitHub"}
                  />
                </div>
                {!isSSHProvider && (
                  <>
                    <div>
                      <label className="block text-sm font-medium mb-2">Base URL</label>
                      <Input
                        value={formBaseUrl}
                        onChange={(e) => setFormBaseUrl(e.target.value)}
                        placeholder={getDefaultBaseUrl(formType)}
                      />
                      <p className="text-xs text-muted-foreground mt-1">
                        For self-hosted instances, enter the base URL
                      </p>
                    </div>
                    <div>
                      <label className="block text-sm font-medium mb-2">Client ID (optional)</label>
                      <Input
                        value={formClientId}
                        onChange={(e) => setFormClientId(e.target.value)}
                        placeholder="OAuth App Client ID"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium mb-2">Client Secret (optional)</label>
                      <Input
                        type="password"
                        value={formClientSecret}
                        onChange={(e) => setFormClientSecret(e.target.value)}
                        placeholder="OAuth App Client Secret"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium mb-2">Bot Token (optional)</label>
                      <Input
                        type="password"
                        value={formBotToken}
                        onChange={(e) => setFormBotToken(e.target.value)}
                        placeholder="Personal Access Token"
                      />
                    </div>
                  </>
                )}
                {isSSHProvider && (
                  <div>
                    <label className="block text-sm font-medium mb-2">SSH Key</label>
                    <select
                      value={formSSHKeyId || ""}
                      onChange={(e) => setFormSSHKeyId(e.target.value ? Number(e.target.value) : null)}
                      className="w-full border border-border rounded px-3 py-2 bg-background"
                    >
                      <option value="">Select an SSH key...</option>
                      {sshKeys.map((key) => (
                        <option key={key.id} value={key.id}>
                          {key.name} ({key.fingerprint.substring(0, 16)}...)
                        </option>
                      ))}
                    </select>
                    {sshKeys.length === 0 && (
                      <p className="text-xs text-muted-foreground mt-1">
                        No SSH keys available. Create one below.
                      </p>
                    )}
                  </div>
                )}
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="isDefault"
                    checked={formIsDefault}
                    onChange={(e) => setFormIsDefault(e.target.checked)}
                  />
                  <label htmlFor="isDefault" className="text-sm">Set as default provider</label>
                </div>
              </div>
              <div className="flex gap-3 mt-6">
                <Button
                  variant="outline"
                  className="flex-1"
                  onClick={() => {
                    setShowAddDialog(false);
                    resetForm();
                  }}
                >
                  Cancel
                </Button>
                <Button
                  className="flex-1"
                  onClick={() => editingProvider ? handleUpdateProvider(editingProvider) : handleAddProvider()}
                  disabled={saving || (isSSHProvider && !formSSHKeyId)}
                >
                  {saving ? "Saving..." : editingProvider ? "Save Changes" : "Add Provider"}
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* SSH Keys Management */}
      <div className="border border-border rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold">SSH Keys</h2>
            <p className="text-sm text-muted-foreground">
              Manage SSH keys for Git authentication
            </p>
          </div>
          <Button onClick={() => setShowSSHKeyDialog(true)}>Add SSH Key</Button>
        </div>

        {sshKeys.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            No SSH keys configured. Add one to use SSH Git providers.
          </div>
        ) : (
          <div className="space-y-4">
            {sshKeys.map((key) => (
              <div
                key={key.id}
                className="flex items-center justify-between p-4 border border-border rounded-lg"
              >
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-lg bg-muted flex items-center justify-center">
                    🔑
                  </div>
                  <div>
                    <h3 className="font-medium">{key.name}</h3>
                    <p className="text-xs text-muted-foreground font-mono">
                      {key.fingerprint}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => copyToClipboard(key.public_key)}
                  >
                    Copy Public Key
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-destructive hover:text-destructive"
                    onClick={() => handleDeleteSSHKey(key.id)}
                  >
                    Delete
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Add SSH Key Dialog */}
        {showSSHKeyDialog && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-background border border-border rounded-lg p-6 w-full max-w-lg">
              {createdSSHKey ? (
                <>
                  <h3 className="text-lg font-semibold mb-4">SSH Key Created</h3>
                  <p className="text-sm text-muted-foreground mb-4">
                    Add this public key to your Git server:
                  </p>
                  <div className="bg-muted p-3 rounded-lg mb-4">
                    <code className="text-xs break-all">{createdSSHKey.public_key}</code>
                  </div>
                  <div className="flex gap-3">
                    <Button
                      variant="outline"
                      className="flex-1"
                      onClick={() => copyToClipboard(createdSSHKey.public_key)}
                    >
                      Copy Public Key
                    </Button>
                    <Button className="flex-1" onClick={resetSSHKeyDialog}>
                      Done
                    </Button>
                  </div>
                </>
              ) : (
                <>
                  <h3 className="text-lg font-semibold mb-4">Add SSH Key</h3>
                  <div className="space-y-4">
                    <div>
                      <label className="block text-sm font-medium mb-2">Name</label>
                      <Input
                        value={newSSHKeyName}
                        onChange={(e) => setNewSSHKeyName(e.target.value)}
                        placeholder="e.g., Production Server Key"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium mb-2">
                        Private Key (optional)
                      </label>
                      <textarea
                        className="w-full px-3 py-2 border border-border rounded-md bg-background font-mono text-xs"
                        rows={6}
                        value={newSSHKeyPrivate}
                        onChange={(e) => setNewSSHKeyPrivate(e.target.value)}
                        placeholder="-----BEGIN OPENSSH PRIVATE KEY-----&#10;...&#10;-----END OPENSSH PRIVATE KEY-----"
                      />
                      <p className="text-xs text-muted-foreground mt-1">
                        Leave empty to generate a new Ed25519 key pair automatically.
                      </p>
                    </div>
                  </div>
                  <div className="flex gap-3 mt-6">
                    <Button variant="outline" className="flex-1" onClick={resetSSHKeyDialog}>
                      Cancel
                    </Button>
                    <Button
                      className="flex-1"
                      onClick={handleCreateSSHKey}
                      disabled={savingSSHKey || !newSSHKeyName}
                    >
                      {savingSSHKey ? "Creating..." : newSSHKeyPrivate ? "Import Key" : "Generate Key"}
                    </Button>
                  </div>
                </>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function BillingSettings() {
  const [loading, setLoading] = useState(true);
  const [overview, setOverview] = useState<BillingOverview | null>(null);
  const [plans, setPlans] = useState<SubscriptionPlan[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [showPlansDialog, setShowPlansDialog] = useState(false);
  const [upgrading, setUpgrading] = useState(false);

  const loadBillingData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [overviewRes, plansRes] = await Promise.all([
        billingApi.getOverview().catch(() => null),
        billingApi.listPlans().catch(() => ({ plans: [] })),
      ]);
      if (overviewRes?.overview) {
        setOverview(overviewRes.overview);
      }
      setPlans(plansRes.plans || []);
    } catch (err) {
      setError("Failed to load billing data");
      console.error("Error loading billing data:", err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadBillingData();
  }, [loadBillingData]);

  const handleUpgrade = async (planName: string) => {
    setUpgrading(true);
    try {
      if (overview) {
        await billingApi.updateSubscription(planName);
      } else {
        await billingApi.createSubscription(planName);
      }
      setShowPlansDialog(false);
      await loadBillingData();
    } catch (err) {
      console.error("Failed to upgrade:", err);
      setError("Failed to upgrade plan");
    } finally {
      setUpgrading(false);
    }
  };

  // Calculate usage percentages
  const getUsagePercent = (current: number, max: number): number => {
    if (max === -1) return 0; // Unlimited
    if (max === 0) return 100;
    return Math.min(100, (current / max) * 100);
  };

  const formatLimit = (value: number): string => {
    return value === -1 ? "Unlimited" : String(value);
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="border border-border rounded-lg p-6 animate-pulse">
          <div className="h-6 bg-muted rounded w-32 mb-4"></div>
          <div className="h-8 bg-muted rounded w-48 mb-2"></div>
          <div className="h-4 bg-muted rounded w-64"></div>
        </div>
      </div>
    );
  }

  if (error && !overview) {
    return (
      <div className="space-y-6">
        <div className="border border-border rounded-lg p-6">
          <p className="text-destructive">{error}</p>
          <Button variant="outline" className="mt-4" onClick={loadBillingData}>
            Retry
          </Button>
        </div>
      </div>
    );
  }

  // If no subscription exists, show setup prompt
  if (!overview) {
    return (
      <div className="space-y-6">
        <div className="border border-border rounded-lg p-6 text-center">
          <h2 className="text-lg font-semibold mb-4">No Active Subscription</h2>
          <p className="text-muted-foreground mb-6">
            Choose a plan to get started with AgentMesh
          </p>
          <Button onClick={() => setShowPlansDialog(true)}>Select a Plan</Button>
        </div>

        {/* Plans Dialog */}
        {showPlansDialog && (
          <PlansDialog
            plans={plans}
            currentPlan={null}
            onSelect={handleUpgrade}
            onClose={() => setShowPlansDialog(false)}
            loading={upgrading}
          />
        )}
      </div>
    );
  }

  const { plan, usage, status, billing_cycle, current_period_end } = overview;

  return (
    <div className="space-y-6">
      {/* Current Plan */}
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">Current Plan</h2>
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-3">
              <h3 className="text-2xl font-bold">{plan?.display_name || plan?.name || "Free"}</h3>
              <span className={`text-xs px-2 py-0.5 rounded ${
                status === "active" ? "bg-green-100 text-green-800" :
                status === "past_due" ? "bg-yellow-100 text-yellow-800" :
                "bg-red-100 text-red-800"
              }`}>
                {status.charAt(0).toUpperCase() + status.slice(1)}
              </span>
            </div>
            <p className="text-muted-foreground">
              {billing_cycle === "yearly" ? "Yearly" : "Monthly"} billing
              {current_period_end && (
                <> · Renews {new Date(current_period_end).toLocaleDateString()}</>
              )}
            </p>
            {plan?.price_per_seat_monthly > 0 && (
              <p className="text-sm text-muted-foreground mt-1">
                ${plan.price_per_seat_monthly}/seat/month
              </p>
            )}
          </div>
          <Button onClick={() => setShowPlansDialog(true)}>
            {plan?.name === "free" ? "Upgrade" : "Change Plan"}
          </Button>
        </div>
      </div>

      {/* Usage */}
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">Usage</h2>
        <div className="space-y-4">
          {/* Pod Minutes */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm">Pod Minutes</span>
              <span className="text-sm font-medium">
                {Math.round(usage.pod_minutes)} / {formatLimit(usage.included_pod_minutes)}
              </span>
            </div>
            <div className="w-full bg-muted rounded-full h-2">
              <div
                className={`h-2 rounded-full ${
                  getUsagePercent(usage.pod_minutes, usage.included_pod_minutes) > 90
                    ? "bg-destructive"
                    : "bg-primary"
                }`}
                style={{ width: `${getUsagePercent(usage.pod_minutes, usage.included_pod_minutes)}%` }}
              ></div>
            </div>
          </div>

          {/* Users */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm">Team Members</span>
              <span className="text-sm font-medium">
                {usage.users} / {formatLimit(usage.max_users)}
              </span>
            </div>
            <div className="w-full bg-muted rounded-full h-2">
              <div
                className={`h-2 rounded-full ${
                  getUsagePercent(usage.users, usage.max_users) > 90 ? "bg-destructive" : "bg-primary"
                }`}
                style={{ width: `${getUsagePercent(usage.users, usage.max_users)}%` }}
              ></div>
            </div>
          </div>

          {/* Runners */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm">Runners</span>
              <span className="text-sm font-medium">
                {usage.runners} / {formatLimit(usage.max_runners)}
              </span>
            </div>
            <div className="w-full bg-muted rounded-full h-2">
              <div
                className={`h-2 rounded-full ${
                  getUsagePercent(usage.runners, usage.max_runners) > 90 ? "bg-destructive" : "bg-primary"
                }`}
                style={{ width: `${getUsagePercent(usage.runners, usage.max_runners)}%` }}
              ></div>
            </div>
          </div>

          {/* Repositories */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm">Repositories</span>
              <span className="text-sm font-medium">
                {usage.repositories} / {formatLimit(usage.max_repositories)}
              </span>
            </div>
            <div className="w-full bg-muted rounded-full h-2">
              <div
                className={`h-2 rounded-full ${
                  getUsagePercent(usage.repositories, usage.max_repositories) > 90
                    ? "bg-destructive"
                    : "bg-primary"
                }`}
                style={{ width: `${getUsagePercent(usage.repositories, usage.max_repositories)}%` }}
              ></div>
            </div>
          </div>
        </div>
      </div>

      {/* Payment Method */}
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">Payment Method</h2>
        <p className="text-muted-foreground">No payment method on file</p>
        <Button variant="outline" className="mt-4">
          Add Payment Method
        </Button>
      </div>

      {/* Plans Dialog */}
      {showPlansDialog && (
        <PlansDialog
          plans={plans}
          currentPlan={plan?.name || null}
          onSelect={handleUpgrade}
          onClose={() => setShowPlansDialog(false)}
          loading={upgrading}
        />
      )}
    </div>
  );
}

// Plans selection dialog component
function PlansDialog({
  plans,
  currentPlan,
  onSelect,
  onClose,
  loading,
}: {
  plans: SubscriptionPlan[];
  currentPlan: string | null;
  onSelect: (planName: string) => void;
  onClose: () => void;
  loading: boolean;
}) {
  const formatLimit = (value: number): string => {
    return value === -1 ? "Unlimited" : String(value);
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border border-border rounded-lg p-6 w-full max-w-4xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-lg font-semibold">Choose a Plan</h3>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            ✕
          </button>
        </div>

        {plans.length === 0 ? (
          <p className="text-center text-muted-foreground py-8">No plans available</p>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {plans.map((plan) => {
              const isCurrent = plan.name === currentPlan;
              return (
                <div
                  key={plan.id}
                  className={`border rounded-lg p-6 ${
                    isCurrent ? "border-primary bg-primary/5" : "border-border"
                  }`}
                >
                  <div className="mb-4">
                    <h4 className="text-xl font-bold">{plan.display_name}</h4>
                    {plan.price_per_seat_monthly > 0 ? (
                      <p className="text-2xl font-bold mt-2">
                        ${plan.price_per_seat_monthly}
                        <span className="text-sm font-normal text-muted-foreground">/seat/month</span>
                      </p>
                    ) : (
                      <p className="text-2xl font-bold mt-2">Free</p>
                    )}
                  </div>

                  <ul className="space-y-2 mb-6 text-sm">
                    <li className="flex items-center gap-2">
                      <span className="text-green-500">✓</span>
                      {formatLimit(plan.included_pod_minutes)} pod minutes
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500">✓</span>
                      {formatLimit(plan.max_users)} team members
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500">✓</span>
                      {formatLimit(plan.max_runners)} runners
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500">✓</span>
                      {formatLimit(plan.max_repositories)} repositories
                    </li>
                  </ul>

                  <Button
                    className="w-full"
                    variant={isCurrent ? "outline" : "default"}
                    disabled={isCurrent || loading}
                    onClick={() => onSelect(plan.name)}
                  >
                    {loading ? "Processing..." : isCurrent ? "Current Plan" : "Select"}
                  </Button>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

// ===== Runners Settings =====

function RunnersSettings() {
  const {
    runners,
    tokens,
    loading,
    error,
    fetchRunners,
    fetchTokens,
    updateRunner,
    deleteRunner,
    regenerateAuthToken,
    createToken,
    revokeToken,
    clearError,
  } = useRunnerStore();

  const [editingRunner, setEditingRunner] = useState<Runner | null>(null);
  const [showTokenDialog, setShowTokenDialog] = useState(false);

  useEffect(() => {
    fetchRunners();
    fetchTokens();
  }, [fetchRunners, fetchTokens]);

  return (
    <div className="space-y-6">
      {error && (
        <div className="bg-destructive/10 border border-destructive text-destructive px-4 py-3 rounded-lg flex items-center justify-between">
          <span>{error}</span>
          <button onClick={clearError} className="text-sm underline">
            Dismiss
          </button>
        </div>
      )}

      {/* Registration Tokens */}
      <TokensPanel
        tokens={tokens}
        loading={loading}
        onCreateToken={createToken}
        onRevokeToken={revokeToken}
        showDialog={showTokenDialog}
        onShowDialog={setShowTokenDialog}
      />

      {/* Runners List */}
      <RunnersPanel
        runners={runners}
        loading={loading}
        onEdit={setEditingRunner}
        onDelete={deleteRunner}
        onRegenerateToken={regenerateAuthToken}
      />

      {/* Edit Runner Dialog */}
      {editingRunner && (
        <EditRunnerDialog
          runner={editingRunner}
          onClose={() => setEditingRunner(null)}
          onSave={async (id, data) => {
            await updateRunner(id, data);
            setEditingRunner(null);
          }}
        />
      )}
    </div>
  );
}

// TokensPanel Component
function TokensPanel({
  tokens,
  loading,
  onCreateToken,
  onRevokeToken,
  showDialog,
  onShowDialog,
}: {
  tokens: RegistrationToken[];
  loading: boolean;
  onCreateToken: (description?: string, maxUses?: number, expiresAt?: string) => Promise<string>;
  onRevokeToken: (id: number) => Promise<void>;
  showDialog: boolean;
  onShowDialog: (show: boolean) => void;
}) {
  const [newTokenDescription, setNewTokenDescription] = useState("");
  const [newTokenMaxUses, setNewTokenMaxUses] = useState<string>("");
  const [newTokenExpires, setNewTokenExpires] = useState<string>("");
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);

  const handleCreateToken = async () => {
    setCreating(true);
    try {
      const maxUses = newTokenMaxUses ? parseInt(newTokenMaxUses, 10) : undefined;
      const expiresAt = newTokenExpires || undefined;
      const token = await onCreateToken(newTokenDescription || undefined, maxUses, expiresAt);
      setCreatedToken(token);
      setNewTokenDescription("");
      setNewTokenMaxUses("");
      setNewTokenExpires("");
    } catch (err) {
      console.error("Failed to create token:", err);
    } finally {
      setCreating(false);
    }
  };

  const handleCloseDialog = () => {
    onShowDialog(false);
    setCreatedToken(null);
    setNewTokenDescription("");
    setNewTokenMaxUses("");
    setNewTokenExpires("");
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString();
  };

  return (
    <div className="border border-border rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">Registration Tokens</h2>
          <p className="text-sm text-muted-foreground">
            Create tokens to register new runners
          </p>
        </div>
        <Button onClick={() => onShowDialog(true)}>Create Token</Button>
      </div>

      {loading ? (
        <div className="text-center py-4 text-muted-foreground">Loading...</div>
      ) : tokens.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          No registration tokens. Create one to register runners.
        </div>
      ) : (
        <div className="space-y-3">
          {tokens.map((token) => (
            <div
              key={token.id}
              className={`flex items-center justify-between p-4 border rounded-lg ${
                token.is_active ? "border-border" : "border-border bg-muted/50 opacity-60"
              }`}
            >
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium">
                    {token.description || `Token #${token.id}`}
                  </span>
                  {!token.is_active && (
                    <span className="text-xs bg-muted px-2 py-0.5 rounded">Revoked</span>
                  )}
                </div>
                <div className="text-sm text-muted-foreground mt-1 space-x-4">
                  <span>Uses: {token.used_count}{token.max_uses ? ` / ${token.max_uses}` : ""}</span>
                  <span>Created: {formatDate(token.created_at)}</span>
                  {token.expires_at && (
                    <span>Expires: {formatDate(token.expires_at)}</span>
                  )}
                </div>
              </div>
              {token.is_active && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-destructive hover:text-destructive"
                  onClick={() => onRevokeToken(token.id)}
                >
                  Revoke
                </Button>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Create Token Dialog */}
      {showDialog && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background border border-border rounded-lg p-6 w-full max-w-md">
            {createdToken ? (
              <>
                <h3 className="text-lg font-semibold mb-4">Token Created</h3>
                <p className="text-sm text-muted-foreground mb-4">
                  Copy this token now. You won&apos;t be able to see it again.
                </p>
                <div className="bg-muted p-3 rounded-lg mb-4 flex items-center justify-between">
                  <code className="text-sm break-all">{createdToken}</code>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => copyToClipboard(createdToken)}
                  >
                    Copy
                  </Button>
                </div>
                <Button className="w-full" onClick={handleCloseDialog}>
                  Done
                </Button>
              </>
            ) : (
              <>
                <h3 className="text-lg font-semibold mb-4">Create Registration Token</h3>
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      Description (optional)
                    </label>
                    <Input
                      value={newTokenDescription}
                      onChange={(e) => setNewTokenDescription(e.target.value)}
                      placeholder="e.g., Dev team runner"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      Max Uses (optional)
                    </label>
                    <Input
                      type="number"
                      value={newTokenMaxUses}
                      onChange={(e) => setNewTokenMaxUses(e.target.value)}
                      placeholder="Leave empty for unlimited"
                      min="1"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      Expires At (optional)
                    </label>
                    <Input
                      type="datetime-local"
                      value={newTokenExpires}
                      onChange={(e) => setNewTokenExpires(e.target.value)}
                    />
                  </div>
                </div>
                <div className="flex gap-3 mt-6">
                  <Button variant="outline" className="flex-1" onClick={handleCloseDialog}>
                    Cancel
                  </Button>
                  <Button className="flex-1" onClick={handleCreateToken} disabled={creating}>
                    {creating ? "Creating..." : "Create Token"}
                  </Button>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// RunnersPanel Component
function RunnersPanel({
  runners,
  loading,
  onEdit,
  onDelete,
  onRegenerateToken,
}: {
  runners: Runner[];
  loading: boolean;
  onEdit: (runner: Runner) => void;
  onDelete: (id: number) => Promise<void>;
  onRegenerateToken: (id: number) => Promise<string>;
}) {
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null);
  const [regeneratedToken, setRegeneratedToken] = useState<{ id: number; token: string } | null>(null);

  const handleDelete = async (id: number) => {
    try {
      await onDelete(id);
      setConfirmDelete(null);
    } catch (err) {
      console.error("Failed to delete runner:", err);
    }
  };

  const handleRegenerateToken = async (id: number) => {
    try {
      const token = await onRegenerateToken(id);
      setRegeneratedToken({ id, token });
    } catch (err) {
      console.error("Failed to regenerate token:", err);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const formatLastSeen = (dateString?: string) => {
    if (!dateString) return "Never";
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffSec = Math.floor(diffMs / 1000);

    if (diffSec < 60) return "Just now";
    if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
    if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
    return date.toLocaleDateString();
  };

  return (
    <div className="border border-border rounded-lg p-6">
      <div className="mb-4">
        <h2 className="text-lg font-semibold">Runners</h2>
        <p className="text-sm text-muted-foreground">
          Manage your self-hosted runners
        </p>
      </div>

      {loading ? (
        <div className="text-center py-4 text-muted-foreground">Loading...</div>
      ) : runners.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          No runners registered. Use a registration token to add runners.
        </div>
      ) : (
        <div className="space-y-3">
          {runners.map((runner) => {
            const statusInfo = getRunnerStatusInfo(runner.status as "online" | "offline" | "maintenance" | "busy");
            return (
              <div
                key={runner.id}
                className={`p-4 border rounded-lg ${
                  runner.is_enabled ? "border-border" : "border-border bg-muted/50"
                }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{runner.node_id}</span>
                      <span
                        className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${statusInfo?.color}`}
                      >
                        <span className={`w-1.5 h-1.5 rounded-full ${statusInfo?.dotColor}`} />
                        {statusInfo?.label}
                      </span>
                      {!runner.is_enabled && (
                        <span className="text-xs bg-yellow-100 text-yellow-800 px-2 py-0.5 rounded">
                          Disabled
                        </span>
                      )}
                    </div>
                    {runner.description && (
                      <p className="text-sm text-muted-foreground mt-1">
                        {runner.description}
                      </p>
                    )}
                    <div className="flex items-center gap-4 text-sm text-muted-foreground mt-2">
                      <span>
                        Pods: {runner.current_pods} / {runner.max_concurrent_pods}
                      </span>
                      {runner.runner_version && <span>v{runner.runner_version}</span>}
                      <span>Last seen: {formatLastSeen(runner.last_heartbeat)}</span>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" onClick={() => onEdit(runner)}>
                      Edit
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleRegenerateToken(runner.id)}
                    >
                      Regenerate Token
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => setConfirmDelete(runner.id)}
                    >
                      Delete
                    </Button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Confirm Delete Dialog */}
      {confirmDelete !== null && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background border border-border rounded-lg p-6 w-full max-w-sm">
            <h3 className="text-lg font-semibold mb-2">Delete Runner</h3>
            <p className="text-muted-foreground mb-4">
              Are you sure you want to delete this runner? This action cannot be undone.
            </p>
            <div className="flex gap-3">
              <Button variant="outline" className="flex-1" onClick={() => setConfirmDelete(null)}>
                Cancel
              </Button>
              <Button
                variant="destructive"
                className="flex-1"
                onClick={() => handleDelete(confirmDelete)}
              >
                Delete
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Regenerated Token Dialog */}
      {regeneratedToken && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background border border-border rounded-lg p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold mb-4">New Auth Token</h3>
            <p className="text-sm text-muted-foreground mb-4">
              Copy this token now. You won&apos;t be able to see it again. The runner will need to be updated with this new token.
            </p>
            <div className="bg-muted p-3 rounded-lg mb-4 flex items-center justify-between">
              <code className="text-sm break-all">{regeneratedToken.token}</code>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => copyToClipboard(regeneratedToken.token)}
              >
                Copy
              </Button>
            </div>
            <Button className="w-full" onClick={() => setRegeneratedToken(null)}>
              Done
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

// Edit Runner Dialog Component
function EditRunnerDialog({
  runner,
  onClose,
  onSave,
}: {
  runner: Runner;
  onClose: () => void;
  onSave: (id: number, data: { description?: string; max_concurrent_pods?: number; is_enabled?: boolean }) => Promise<void>;
}) {
  const [description, setDescription] = useState(runner.description || "");
  const [maxPods, setMaxPods] = useState(runner.max_concurrent_pods.toString());
  const [isEnabled, setIsEnabled] = useState(runner.is_enabled);
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    try {
      await onSave(runner.id, {
        description: description || undefined,
        max_concurrent_pods: parseInt(maxPods, 10),
        is_enabled: isEnabled,
      });
    } catch (err) {
      console.error("Failed to save runner:", err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border border-border rounded-lg p-6 w-full max-w-md">
        <h3 className="text-lg font-semibold mb-4">Edit Runner</h3>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">Node ID</label>
            <Input value={runner.node_id} disabled />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">Description</label>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Runner description"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">
              Max Concurrent Pods
            </label>
            <Input
              type="number"
              value={maxPods}
              onChange={(e) => setMaxPods(e.target.value)}
              min="1"
            />
          </div>
          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">Enabled</label>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                className="sr-only peer"
                checked={isEnabled}
                onChange={(e) => setIsEnabled(e.target.checked)}
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
            </label>
          </div>
        </div>
        <div className="flex gap-3 mt-6">
          <Button variant="outline" className="flex-1" onClick={onClose}>
            Cancel
          </Button>
          <Button className="flex-1" onClick={handleSave} disabled={saving}>
            {saving ? "Saving..." : "Save Changes"}
          </Button>
        </div>
      </div>
    </div>
  );
}

// ===== Notifications Settings =====

function NotificationsSettings() {
  return (
    <div className="space-y-6">
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">Push Notifications</h2>
        <p className="text-sm text-muted-foreground mb-6">
          Configure push notifications for important events like pod status changes,
          ticket assignments, and runner alerts.
        </p>
        <NotificationSettings />
      </div>
    </div>
  );
}
