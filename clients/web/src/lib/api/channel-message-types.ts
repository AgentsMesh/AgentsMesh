export interface InlineStyle {
  bold?: boolean;
  italic?: boolean;
  strike?: boolean;
  code?: boolean;
}

export interface InlineElement {
  type: "text" | "mention" | "link" | "linebreak";
  text?: string;
  style?: InlineStyle;
  bold?: boolean;
  italic?: boolean;
  strike?: boolean;
  code?: boolean;
  entity_type?: "pod" | "user" | "ticket" | "channel";
  entity_key?: string;
  display?: string;
  url?: string;
}

export interface Block {
  type: "paragraph" | "heading" | "code_block" | "quote" | "list";
  elements?: InlineElement[];
  children?: Block[];
  level?: number;
  language?: string;
  text?: string;
  ordered?: boolean;
  items?: Block[][];
}

export interface MessageContent {
  schema_version?: number;
  kind: string;
  blocks?: Block[];
  attachment_key?: string;
}

export interface MessageMentions {
  pods?: string[];
  users?: number[];
  channel?: boolean;
}

export interface MentionRefInput {
  entity_type: "pod" | "user";
  entity_key: string;
}

export interface MessageSendPayload {
  source: string;
  mentions?: Record<string, MentionRefInput>;
  attachment_key?: string;
}

export interface MessageEditPayload {
  source: string;
  mentions?: Record<string, MentionRefInput>;
}
