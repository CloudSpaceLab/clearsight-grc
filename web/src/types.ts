export type AttentionItem = {
  id: string;
  type: string;
  title: string;
  why_now: string;
  scope: string;
  state: string;
  evidence: string;
  owner: string;
  due_at: string;
  primary_action: string;
};

export type CaptureRequest = {
  id: string;
  title: string;
  purpose: string;
  why_you: string;
  status: string;
  sensitivity: string;
  estimated_minutes: number;
  deadline: string;
  known_facts: Record<string, string>;
  fields: Array<{
    id: string;
    label: string;
    type: string;
    required: boolean;
    description?: string;
    options?: string[];
  }>;
  version: number;
};

export type AuthorityResolution = {
  principal: {
    id: string;
    display_name: string;
    kind: string;
    role: string;
  };
  rule_id: string;
  policy_version: string;
  explanation: string;
};
