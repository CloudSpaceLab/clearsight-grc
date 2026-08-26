export type VendorLinkTargetType = "PROGRAM" | "MATTER";
export type VendorRelationshipLinkState = "ACTIVE" | "ENDED";

export type VendorRelationshipLink = {
  id: string;
  tenant_id?: string;
  legal_entity_id?: string;
  relationship_id: string;
  target_type: VendorLinkTargetType;
  target_id: string;
  purpose_code: string;
  purpose_label: string;
  state: VendorRelationshipLinkState;
  created_by?: string;
  ended_by?: string;
  end_reason?: string;
  version: number;
  created_at?: string;
  updated_at?: string;
  ended_at?: string;
};

export type VendorRelationshipLinkPage = {
  items: VendorRelationshipLink[];
  next_cursor?: string;
};

export type VendorRelationshipLinkQuery = ({ target_type: VendorLinkTargetType; target_id: string; relationship_id?: never } | { relationship_id: string; target_type?: never; target_id?: never }) & { cursor?: string; limit?: number };

export type LinkVendorRelationshipInput = {
  target_type: VendorLinkTargetType;
  target_id: string;
  purpose_code: string;
  purpose_label: string;
};

export type EndVendorRelationshipLinkInput = {
  expected_version: number;
  reason: string;
};
