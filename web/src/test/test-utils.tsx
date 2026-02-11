import { ReactElement, ReactNode } from "react";
import { render, RenderOptions } from "@testing-library/react";
import { NextIntlClientProvider } from "next-intl";
import commonMessages from "@/messages/en/common.json";
import authMessages from "@/messages/en/auth.json";
import landingMessages from "@/messages/en/landing.json";
import appMessages from "@/messages/en/app.json";
import settingsMessages from "@/messages/en/settings.json";
import ideMessages from "@/messages/en/ide.json";
import repositoriesMessages from "@/messages/en/repositories.json";
import runnersMessages from "@/messages/en/runners.json";
import docsMessages from "@/messages/en/docs.json";
import contentMessages from "@/messages/en/content.json";

// Mock translations for testing
const mockTranslations = {
  ...commonMessages,
  ...authMessages,
  ...landingMessages,
  ...appMessages,
  ...settingsMessages,
  ...ideMessages,
  ...repositoriesMessages,
  ...runnersMessages,
  ...docsMessages,
  ...contentMessages,
};

// Wrapper component that provides all necessary providers for testing
function AllProviders({ children }: { children: ReactNode }) {
  return (
    <NextIntlClientProvider locale="en" messages={mockTranslations}>
      {children}
    </NextIntlClientProvider>
  );
}

// Custom render function that wraps components with providers
const customRender = (
  ui: ReactElement,
  options?: Omit<RenderOptions, "wrapper">
) => render(ui, { wrapper: AllProviders, ...options });

// Re-export everything from testing-library
export * from "@testing-library/react";

// Override render method
export { customRender as render };
