
export interface KanbanTicket {
  id: string;
  title: string;
  agent: string;
  status: "backlog" | "in_progress" | "done";
  color: string;
  executing?: boolean;
}

export interface TerminalLine {
  text: string;
  type: "command" | "output" | "success" | "info";
}

export interface TerminalPane {
  id: string;
  agent: string;
  ticketId: string;
  lines: TerminalLine[];
  active: boolean;
}

export interface DemoFrame {
  time: number;
  tickets: KanbanTicket[];
  terminals: TerminalPane[];
  showTerminals: boolean;
}
