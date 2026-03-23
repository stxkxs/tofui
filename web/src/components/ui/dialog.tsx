import {
  type ReactNode,
  type MouseEvent,
  useEffect,
  useCallback,
  useRef,
} from "react";
import { cn } from "@/lib/utils";

interface DialogProps {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
}

const FOCUSABLE_SELECTOR =
  'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

export function Dialog({ open, onClose, children }: DialogProps) {
  const contentRef = useRef<HTMLDivElement>(null);

  const handleEscape = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    },
    [onClose]
  );

  const handleFocusTrap = useCallback((e: KeyboardEvent) => {
    if (e.key !== "Tab" || !contentRef.current) return;

    const focusable = contentRef.current.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR);
    if (focusable.length === 0) return;

    const first = focusable[0];
    const last = focusable[focusable.length - 1];

    if (e.shiftKey) {
      if (document.activeElement === first) {
        e.preventDefault();
        last.focus();
      }
    } else {
      if (document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }
  }, []);

  useEffect(() => {
    if (open) {
      document.addEventListener("keydown", handleEscape);
      document.addEventListener("keydown", handleFocusTrap);
      document.body.style.overflow = "hidden";

      requestAnimationFrame(() => {
        if (contentRef.current) {
          const first = contentRef.current.querySelector<HTMLElement>(FOCUSABLE_SELECTOR);
          first?.focus();
        }
      });
    }
    return () => {
      document.removeEventListener("keydown", handleEscape);
      document.removeEventListener("keydown", handleFocusTrap);
      document.body.style.overflow = "";
    };
  }, [open, handleEscape, handleFocusTrap]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby="dialog-title"
    >
      <div
        className="fixed inset-0 bg-black/70 backdrop-blur-md transition-opacity duration-200"
        aria-hidden="true"
        onClick={onClose}
      />
      <div ref={contentRef} className="relative z-50 w-full max-w-lg mx-4 animate-fade-up">
        {children}
      </div>
    </div>
  );
}

export function DialogContent({
  className,
  children,
  ...props
}: {
  className?: string;
  children: ReactNode;
}) {
  return (
    <div
      className={cn(
        "rounded-xl border border-border/60 bg-card/95 backdrop-blur-sm p-6 shadow-2xl shadow-black/40",
        className
      )}
      onClick={(e: MouseEvent) => e.stopPropagation()}
      {...props}
    >
      {children}
    </div>
  );
}

export function DialogHeader({ children }: { children: ReactNode }) {
  return <div className="mb-5">{children}</div>;
}

export function DialogTitle({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <h2
      id="dialog-title"
      className={cn("text-lg font-semibold text-foreground", className)}
    >
      {children}
    </h2>
  );
}

export function DialogDescription({
  children,
}: {
  children: ReactNode;
}) {
  return <p className="text-sm text-muted-foreground mt-1.5">{children}</p>;
}
