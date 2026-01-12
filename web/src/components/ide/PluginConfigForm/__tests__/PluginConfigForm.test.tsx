import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PluginConfigForm } from "../index";
import type { PluginCapability } from "@/lib/api/runner";

// Mock plugins for testing
const mockPlugins: PluginCapability[] = [
  {
    name: "test-plugin",
    version: "1.0.0",
    description: "Test Plugin Description",
    supported_agents: ["test-agent"],
    ui: {
      configurable: true,
      fields: [
        {
          name: "enabled",
          type: "boolean",
          label: "Enable Feature",
          default: true,
          description: "Toggle this feature on/off",
        },
        {
          name: "name",
          type: "string",
          label: "Name",
          placeholder: "Enter name",
          required: true,
        },
        {
          name: "secret_key",
          type: "secret",
          label: "API Key",
          placeholder: "Enter API key",
        },
        {
          name: "count",
          type: "number",
          label: "Count",
          default: 10,
          min: 1,
          max: 100,
        },
        {
          name: "mode",
          type: "select",
          label: "Mode",
          default: "auto",
          options: [
            { value: "auto", label: "Auto" },
            { value: "manual", label: "Manual" },
          ],
        },
      ],
    },
  },
];

describe("PluginConfigForm", () => {
  describe("rendering", () => {
    it("should render null when plugins is empty", () => {
      const { container } = render(
        <PluginConfigForm plugins={[]} values={{}} onChange={vi.fn()} />
      );
      expect(container.firstChild).toBeNull();
    });

    it("should render plugin description", () => {
      render(
        <PluginConfigForm
          plugins={mockPlugins}
          values={{}}
          onChange={vi.fn()}
        />
      );
      expect(screen.getByText("Test Plugin Description")).toBeInTheDocument();
    });

    it("should render all field types", () => {
      render(
        <PluginConfigForm
          plugins={mockPlugins}
          values={{}}
          onChange={vi.fn()}
        />
      );

      // Check boolean field (checkbox)
      expect(screen.getByLabelText("Enable Feature")).toBeInTheDocument();

      // Check string field (text input)
      expect(screen.getByLabelText(/Name/)).toBeInTheDocument();

      // Check secret field (password input)
      expect(screen.getByLabelText("API Key")).toBeInTheDocument();

      // Check number field
      expect(screen.getByLabelText(/Count/)).toBeInTheDocument();

      // Check select field
      expect(screen.getByLabelText(/Mode/)).toBeInTheDocument();
    });

    it("should use default values when no values provided", () => {
      render(
        <PluginConfigForm
          plugins={mockPlugins}
          values={{}}
          onChange={vi.fn()}
        />
      );

      // Boolean field should be checked (default: true)
      const checkbox = screen.getByLabelText("Enable Feature") as HTMLInputElement;
      expect(checkbox.checked).toBe(true);

      // Number field should have default value
      const numberInput = screen.getByLabelText(/Count/) as HTMLInputElement;
      expect(numberInput.value).toBe("10");

      // Select field should have default value
      const select = screen.getByLabelText(/Mode/) as HTMLSelectElement;
      expect(select.value).toBe("auto");
    });

    it("should use provided values over defaults", () => {
      render(
        <PluginConfigForm
          plugins={mockPlugins}
          values={{
            "test-plugin.enabled": false,
            "test-plugin.count": 50,
            "test-plugin.mode": "manual",
          }}
          onChange={vi.fn()}
        />
      );

      // Boolean field should be unchecked
      const checkbox = screen.getByLabelText("Enable Feature") as HTMLInputElement;
      expect(checkbox.checked).toBe(false);

      // Number field should have provided value
      const numberInput = screen.getByLabelText(/Count/) as HTMLInputElement;
      expect(numberInput.value).toBe("50");

      // Select field should have provided value
      const select = screen.getByLabelText(/Mode/) as HTMLSelectElement;
      expect(select.value).toBe("manual");
    });

    it("should fall back to plugin name when no description", () => {
      const pluginWithoutDescription: PluginCapability[] = [
        {
          name: "no-desc-plugin",
          version: "1.0.0",
          description: "",
          supported_agents: [],
          ui: {
            configurable: true,
            fields: [
              { name: "test", type: "string", label: "Test" },
            ],
          },
        },
      ];

      render(
        <PluginConfigForm
          plugins={pluginWithoutDescription}
          values={{}}
          onChange={vi.fn()}
        />
      );
      expect(screen.getByText("no-desc-plugin")).toBeInTheDocument();
    });
  });

  describe("onChange handler", () => {
    it("should call onChange when boolean field changes", () => {
      const handleChange = vi.fn();
      render(
        <PluginConfigForm
          plugins={mockPlugins}
          values={{}}
          onChange={handleChange}
        />
      );

      const checkbox = screen.getByLabelText("Enable Feature");
      fireEvent.click(checkbox);

      expect(handleChange).toHaveBeenCalledWith(
        "test-plugin",
        "enabled",
        false
      );
    });

    it("should call onChange when string field changes", () => {
      const handleChange = vi.fn();
      render(
        <PluginConfigForm
          plugins={mockPlugins}
          values={{}}
          onChange={handleChange}
        />
      );

      const input = screen.getByLabelText(/Name/);
      fireEvent.change(input, { target: { value: "test value" } });

      expect(handleChange).toHaveBeenCalledWith(
        "test-plugin",
        "name",
        "test value"
      );
    });

    it("should call onChange when number field changes", () => {
      const handleChange = vi.fn();
      render(
        <PluginConfigForm
          plugins={mockPlugins}
          values={{}}
          onChange={handleChange}
        />
      );

      const input = screen.getByLabelText(/Count/);
      fireEvent.change(input, { target: { value: "25" } });

      expect(handleChange).toHaveBeenCalledWith(
        "test-plugin",
        "count",
        25
      );
    });

    it("should call onChange when select field changes", () => {
      const handleChange = vi.fn();
      render(
        <PluginConfigForm
          plugins={mockPlugins}
          values={{}}
          onChange={handleChange}
        />
      );

      const select = screen.getByLabelText(/Mode/);
      fireEvent.change(select, { target: { value: "manual" } });

      expect(handleChange).toHaveBeenCalledWith(
        "test-plugin",
        "mode",
        "manual"
      );
    });

    it("should call onChange when secret field changes", () => {
      const handleChange = vi.fn();
      render(
        <PluginConfigForm
          plugins={mockPlugins}
          values={{}}
          onChange={handleChange}
        />
      );

      const input = screen.getByLabelText("API Key");
      fireEvent.change(input, { target: { value: "secret123" } });

      expect(handleChange).toHaveBeenCalledWith(
        "test-plugin",
        "secret_key",
        "secret123"
      );
    });
  });

  describe("multiple plugins", () => {
    it("should render multiple plugins", () => {
      const multiplePlugins: PluginCapability[] = [
        {
          name: "plugin-1",
          version: "1.0.0",
          description: "First Plugin",
          supported_agents: [],
          ui: {
            configurable: true,
            fields: [{ name: "field1", type: "string", label: "Field 1" }],
          },
        },
        {
          name: "plugin-2",
          version: "1.0.0",
          description: "Second Plugin",
          supported_agents: [],
          ui: {
            configurable: true,
            fields: [{ name: "field2", type: "string", label: "Field 2" }],
          },
        },
      ];

      render(
        <PluginConfigForm
          plugins={multiplePlugins}
          values={{}}
          onChange={vi.fn()}
        />
      );

      expect(screen.getByText("First Plugin")).toBeInTheDocument();
      expect(screen.getByText("Second Plugin")).toBeInTheDocument();
      expect(screen.getByLabelText("Field 1")).toBeInTheDocument();
      expect(screen.getByLabelText("Field 2")).toBeInTheDocument();
    });
  });

  describe("unknown field type", () => {
    it("should render fallback for unknown field type", () => {
      const pluginWithUnknownField: PluginCapability[] = [
        {
          name: "unknown-field-plugin",
          version: "1.0.0",
          description: "Unknown Field Plugin",
          supported_agents: [],
          ui: {
            configurable: true,
            fields: [{ name: "unknown", type: "custom-type" as "string", label: "Unknown" }],
          },
        },
      ];

      render(
        <PluginConfigForm
          plugins={pluginWithUnknownField}
          values={{}}
          onChange={vi.fn()}
        />
      );

      expect(screen.getByText(/Unknown field type/)).toBeInTheDocument();
    });
  });

  describe("accessibility", () => {
    it("should have proper ARIA attributes", () => {
      render(
        <PluginConfigForm
          plugins={mockPlugins}
          values={{}}
          onChange={vi.fn()}
        />
      );

      // Check that required fields have aria-required
      const nameInput = screen.getByLabelText(/Name/);
      expect(nameInput).toHaveAttribute("aria-required", "true");

      // Check description association
      const enabledCheckbox = screen.getByLabelText("Enable Feature");
      expect(enabledCheckbox).toHaveAttribute("aria-describedby");
    });
  });
});
