import type { ReactNode } from "react";

type AsyncStatusTone = "info" | "success" | "warning" | "progress" | "error";

type AsyncStatusMessageProps = {
  children: ReactNode;
  tone?: AsyncStatusTone;
  id?: string;
  className?: string;
  testId?: string;
};

export function AsyncStatusMessage({ children, tone = "info", id, className, testId }: AsyncStatusMessageProps) {
  const isError = tone === "error";
  return (
    <p
      id={id}
      className={className}
      role={isError ? "alert" : "status"}
      aria-live={isError ? "assertive" : "polite"}
      data-testid={testId}
    >
      {children}
    </p>
  );
}
