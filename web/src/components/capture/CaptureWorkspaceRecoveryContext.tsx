import { createContext, useContext, type ReactNode } from "react";

export type CaptureWorkspaceRecoveryUI = {
  initialPage: number;
  filesToReselect: readonly string[];
  onPageChange: (page: number) => void;
};

const CaptureWorkspaceRecoveryContext = createContext<CaptureWorkspaceRecoveryUI | null>(null);

export function CaptureWorkspaceRecoveryProvider({ value, children }: { value: CaptureWorkspaceRecoveryUI; children: ReactNode }) {
  return <CaptureWorkspaceRecoveryContext.Provider value={value}>{children}</CaptureWorkspaceRecoveryContext.Provider>;
}

export function useCaptureWorkspaceRecoveryUI(): CaptureWorkspaceRecoveryUI | null {
  return useContext(CaptureWorkspaceRecoveryContext);
}
