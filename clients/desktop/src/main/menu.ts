import { Menu, shell, type MenuItemConstructorOptions } from "electron";

interface MenuDeps {
  logsDir: string;
  onNewWindow: () => void;
}

export function buildMenu(deps: MenuDeps): void {
  const isMac = process.platform === "darwin";
  const template: MenuItemConstructorOptions[] = [
    ...(isMac ? [{ role: "appMenu" } as MenuItemConstructorOptions] : []),
    {
      label: "File",
      submenu: [
        { label: "New Window", accelerator: "CmdOrCtrl+N", click: deps.onNewWindow },
        { type: "separator" },
        isMac ? { role: "close" } : { role: "quit" },
      ],
    },
    { role: "editMenu" },
    { role: "viewMenu" },
    { role: "windowMenu" },
    {
      role: "help",
      submenu: [{ label: "Open Logs", click: () => void shell.openPath(deps.logsDir) }],
    },
  ];
  Menu.setApplicationMenu(Menu.buildFromTemplate(template));
}
