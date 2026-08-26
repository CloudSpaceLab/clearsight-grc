export type VendorStatus = "ACTIVE" | "INACTIVE";
export type VendorRelationshipStatus = "PROPOSED" | "UNDER_REVIEW" | "ACTIVE" | "RESTRICTED" | "SUSPENDED" | "EXITING" | "TERMINATED";
export type VendorCriticality = "STANDARD" | "IMPORTANT" | "CRITICAL";
export type VendorPrivacyRole = "NONE" | "PROCESSOR" | "JOINT_CONTROLLER";
export type VendorBrandState = "APPROVED_LOGO" | "WEBSITE_ICON" | "PENDING" | "UNAVAILABLE";
export type VendorBrandSource = "APPROVED_UPLOAD" | "VENDOR_WEBSITE";

export type VendorBrandPresentation = {
  state: VendorBrandState;
  source?: VendorBrandSource;
  asset_token?: string;
  version: number;
  event_version: number;
  updated_at?: string;
};

export type Vendor = {
  id: string;
  tenant_id: string;
  legal_name: string;
  trading_name?: string;
  registration_ref?: string;
  jurisdiction?: string;
  website_domain?: string;
  source_id?: string;
  external_ref?: string;
  status: VendorStatus;
  created_at: string;
  updated_at: string;
  version: number;
};

export type VendorRelationship = {
  id: string;
  tenant_id: string;
  legal_entity_id: string;
  vendor_id: string;
  service_name: string;
  business_owner_principal_id: string;
  criticality: VendorCriticality;
  privacy_role: VendorPrivacyRole;
  status: VendorRelationshipStatus;
  effective_from?: string;
  renewal_at?: string;
  source_id?: string;
  external_ref?: string;
  created_at: string;
  updated_at: string;
  version: number;
};

export type VendorRelationshipAggregate = { vendor: Vendor; relationship: VendorRelationship; brand?: VendorBrandPresentation };
export type VendorRelationshipPage = { items: VendorRelationshipAggregate[]; next_cursor?: string };
export type VendorIdentityPresentation = { vendor: Vendor; brand: VendorBrandPresentation };

export type CreateVendorRelationshipInput = {
  existing_relationship_id?: string;
  legal_name: string;
  trading_name?: string;
  registration_ref?: string;
  jurisdiction?: string;
  source_id?: string;
  external_ref?: string;
  service_name: string;
  criticality: VendorCriticality;
  privacy_role: VendorPrivacyRole;
  effective_from?: string;
  renewal_at?: string;
};

export type UpdateVendorRelationshipInput = Pick<CreateVendorRelationshipInput, "service_name" | "criticality" | "privacy_role" | "effective_from" | "renewal_at"> &
  { expected_version: number };

export type UpdateVendorIdentityInput = {
  expected_version: number;
  legal_name: string;
  trading_name?: string;
  registration_ref?: string;
  jurisdiction?: string;
  website_domain?: string;
};
