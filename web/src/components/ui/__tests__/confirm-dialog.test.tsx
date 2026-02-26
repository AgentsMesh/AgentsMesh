import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@/test/test-utils";
import { ConfirmDialog, useConfirmDialog } from "../confirm-dialog";
import { Button } from "../button";
import { renderHook } from "@testing-library/react";

describe("ConfirmDialog", () => {
  const defaultProps = {
    open: true,
    onOpenChange: vi.fn(),
    title: "Confirm Action",
    onConfirm: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders title and description", () => {
    render(
      <ConfirmDialog
        {...defaultProps}
        description="Are you sure you want to proceed?"
      />
    );

    expect(screen.getByText("Confirm Action")).toBeInTheDocument();
    expect(
      screen.getByText("Are you sure you want to proceed?")
    ).toBeInTheDocument();
  });

  it("renders default button text", () => {
    render(<ConfirmDialog {...defaultProps} />);

    expect(screen.getByText("Confirm")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("renders custom button text", () => {
    render(
      <ConfirmDialog
        {...defaultProps}
        confirmText="Delete"
        cancelText="Keep"
      />
    );

    expect(screen.getByText("Delete")).toBeInTheDocument();
    expect(screen.getByText("Keep")).toBeInTheDocument();
  });

  it("calls onConfirm when confirm button is clicked", async () => {
    const onConfirm = vi.fn();
    render(<ConfirmDialog {...defaultProps} onConfirm={onConfirm} />);

    fireEvent.click(screen.getByText("Confirm"));

    await waitFor(() => {
      expect(onConfirm).toHaveBeenCalledTimes(1);
    });
  });

  it("calls onOpenChange(false) when cancel button is clicked", () => {
    const onOpenChange = vi.fn();
    render(<ConfirmDialog {...defaultProps} onOpenChange={onOpenChange} />);

    fireEvent.click(screen.getByText("Cancel"));

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("closes dialog after successful confirm", async () => {
    const onOpenChange = vi.fn();
    const onConfirm = vi.fn().mockResolvedValue(undefined);
    render(
      <ConfirmDialog
        {...defaultProps}
        onConfirm={onConfirm}
        onOpenChange={onOpenChange}
      />
    );

    fireEvent.click(screen.getByText("Confirm"));

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });

  it("shows loading state during async confirm", async () => {
    let resolveConfirm: () => void;
    const onConfirm = vi.fn().mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveConfirm = resolve;
        })
    );

    render(<ConfirmDialog {...defaultProps} onConfirm={onConfirm} />);

    fireEvent.click(screen.getByText("Confirm"));

    await waitFor(() => {
      expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    // Resolve the promise
    act(() => {
      resolveConfirm();
    });

    await waitFor(() => {
      expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
    });
  });

  it("disables buttons during loading", async () => {
    let resolveConfirm: () => void;
    const onConfirm = vi.fn().mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveConfirm = resolve;
        })
    );

    render(<ConfirmDialog {...defaultProps} onConfirm={onConfirm} />);

    fireEvent.click(screen.getByText("Confirm"));

    await waitFor(() => {
      const cancelButton = screen.getByText("Cancel");
      expect(cancelButton).toBeDisabled();
    });

    act(() => {
      resolveConfirm();
    });
  });

  it("shows icon by default", () => {
    render(<ConfirmDialog {...defaultProps} />);

    // Dialog is portaled to document.body, so query from there
    const iconContainer = document.body.querySelector(".w-12.h-12.rounded-full");
    expect(iconContainer).toBeInTheDocument();
  });

  it("hides icon when showIcon is false", () => {
    const { container } = render(
      <ConfirmDialog {...defaultProps} showIcon={false} />
    );

    const iconContainer = container.querySelector(".w-12.h-12.rounded-full");
    expect(iconContainer).not.toBeInTheDocument();
  });

  it("renders destructive variant styling", () => {
    render(
      <ConfirmDialog {...defaultProps} variant="destructive" />
    );

    // Dialog is portaled to document.body, so query from there
    const iconContainer = document.body.querySelector(".w-12.h-12.rounded-full");
    expect(iconContainer).toHaveClass("text-destructive");
  });

  it("renders children content", () => {
    render(
      <ConfirmDialog {...defaultProps}>
        <div data-testid="custom-content">Custom content here</div>
      </ConfirmDialog>
    );

    expect(screen.getByTestId("custom-content")).toBeInTheDocument();
    expect(screen.getByText("Custom content here")).toBeInTheDocument();
  });

  it("does not render when open is false", () => {
    render(<ConfirmDialog {...defaultProps} open={false} />);

    expect(screen.queryByText("Confirm Action")).not.toBeInTheDocument();
  });

  it("disables confirm button when confirmDisabled is true", () => {
    render(<ConfirmDialog {...defaultProps} confirmDisabled />);

    const confirmButton = screen.getByText("Confirm");
    expect(confirmButton).toBeDisabled();
  });
});

describe("useConfirmDialog", () => {
  it("returns dialogProps and confirm function with default options", () => {
    const { result } = renderHook(() =>
      useConfirmDialog({
        title: "Test Title",
        description: "Test Description",
      })
    );

    expect(result.current.dialogProps).toBeDefined();
    expect(result.current.dialogProps.title).toBe("Test Title");
    expect(result.current.dialogProps.description).toBe("Test Description");
    expect(result.current.confirm).toBeInstanceOf(Function);
    expect(result.current.isOpen).toBe(false);
  });

  it("works without default options", () => {
    const { result } = renderHook(() => useConfirmDialog());

    expect(result.current.dialogProps).toBeDefined();
    expect(result.current.confirm).toBeInstanceOf(Function);
    expect(result.current.isOpen).toBe(false);
  });

  it("accepts dynamic options when calling confirm", async () => {
    const { result } = renderHook(() => useConfirmDialog());

    act(() => {
      result.current.confirm({
        title: "Dynamic Title",
        description: "Dynamic Description",
        variant: "destructive",
      });
    });

    expect(result.current.isOpen).toBe(true);
    expect(result.current.dialogProps.title).toBe("Dynamic Title");
    expect(result.current.dialogProps.description).toBe("Dynamic Description");
    expect(result.current.dialogProps.variant).toBe("destructive");
  });

  it("opens dialog when confirm is called", async () => {
    const { result } = renderHook(() =>
      useConfirmDialog({
        title: "Test Title",
      })
    );

    expect(result.current.isOpen).toBe(false);

    act(() => {
      result.current.confirm();
    });

    expect(result.current.isOpen).toBe(true);
  });

  it("returns true when confirmed", async () => {
    const { result } = renderHook(() =>
      useConfirmDialog({
        title: "Test Title",
      })
    );

    let confirmResult: boolean | undefined;

    act(() => {
      result.current.confirm().then((value) => {
        confirmResult = value;
      });
    });

    // Simulate confirm
    act(() => {
      result.current.dialogProps.onConfirm();
    });

    await waitFor(() => {
      expect(confirmResult).toBe(true);
    });
  });

  it("returns false when cancelled", async () => {
    const { result } = renderHook(() =>
      useConfirmDialog({
        title: "Test Title",
      })
    );

    let confirmResult: boolean | undefined;

    act(() => {
      result.current.confirm().then((value) => {
        confirmResult = value;
      });
    });

    // Simulate cancel
    act(() => {
      result.current.dialogProps.onOpenChange(false);
    });

    await waitFor(() => {
      expect(confirmResult).toBe(false);
    });
  });
});

// Integration test with a component
describe("ConfirmDialog Integration", () => {
  function TestComponent({ onDelete }: { onDelete: () => Promise<void> }) {
    const { dialogProps, confirm } = useConfirmDialog({
      title: "Delete Item",
      description: "This cannot be undone.",
      variant: "destructive",
      confirmText: "Delete",
    });

    const handleDelete = async () => {
      const confirmed = await confirm();
      if (confirmed) {
        await onDelete();
      }
    };

    return (
      <>
        <Button onClick={handleDelete}>Delete Item</Button>
        <ConfirmDialog {...dialogProps} />
      </>
    );
  }

  it("integrates with useConfirmDialog hook", async () => {
    const onDelete = vi.fn().mockResolvedValue(undefined);
    render(<TestComponent onDelete={onDelete} />);

    // Click delete button
    fireEvent.click(screen.getByText("Delete Item"));

    // Dialog should open
    await waitFor(() => {
      expect(screen.getByText("Delete Item", { selector: "h3" })).toBeInTheDocument();
    });

    // Click confirm
    fireEvent.click(screen.getByText("Delete"));

    // onDelete should be called
    await waitFor(() => {
      expect(onDelete).toHaveBeenCalledTimes(1);
    });
  });

  it("does not call onDelete when cancelled", async () => {
    const onDelete = vi.fn();
    render(<TestComponent onDelete={onDelete} />);

    // Click delete button
    fireEvent.click(screen.getByText("Delete Item"));

    // Dialog should open
    await waitFor(() => {
      expect(screen.getByText("Delete Item", { selector: "h3" })).toBeInTheDocument();
    });

    // Click cancel
    fireEvent.click(screen.getByText("Cancel"));

    // onDelete should not be called
    expect(onDelete).not.toHaveBeenCalled();
  });
});
