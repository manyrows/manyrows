export interface Account {
  id: string;
  email: string;
  name: string;
  validatedAt?: Date;
  language?: string;
  totpEnabled?: boolean;
  isSuper?: boolean;
}

export interface Workspace {
  id: string;
  name: string;
  slug: string;
  status: string;
  createdAt: Date;
  role: string;
  products: Product[];
  cookieDomain?: string | null;
  // First-boot setup checklist state. Both nullable; non-null means
  // "done". UI uses these to render the workspace-home checklist card
  // until either dismissed or all items complete.
  setupChecklistDismissedAt?: string | null;
  setupTestEmailSentAt?: string | null;
}

export interface Product {
  id: string;
  workspaceId: string;
  name: string;
  createdAt: string;
  updatedAt: string;
  createdBy?: string;
}

export interface Permission {
  id: string;
  productId: string;
  name: string;
  slug: string;
  group?: string;
  createdAt?: string;
  updatedAt?: string;
}

export type AppType = "dev" | "staging" | "prod";

export interface App {
  id: string;
  productId: string;
  type: AppType | string;
  // productName is the parent product's name, populated by the server
  // on every read. There is no freeform apps.name anymore - the visible
  // app label is computed from product + env type via appDisplayName.
  productName?: string;
  userPoolId?: string;
  // userPoolName is the identity pool's display name, populated by the
  // server via subquery. Admin views surface it as the "Users here =
  // Users in pool X" signal (especially when multiple apps share a
  // pool for SSO).
  userPoolName?: string;
  enabled?: boolean;
  appUrl?: string;
  authDomain?: string;
  createdAt: string;
  updatedAt: string;
}

// appTypeLabel maps an app's type to the human-readable label.
// Sites that want to show "what kind of app is this" reach for this
// helper instead of hand-rolling the switch.
export function appTypeLabel(app: { type?: AppType | string } | null | undefined): string {
  switch (app?.type) {
    case "prod": return "Production";
    case "staging": return "Staging";
    case "dev": return "Development";
    default: return app?.type ? String(app.type) : "";
  }
}

// appDisplayName composes the visible app label from the parent
// product's name + env-type suffix ("Drum Kingdom (Staging)"). Prod
// drops the suffix - one prod per product, no ambiguity to resolve.
export function appDisplayName(
  app: { type?: AppType | string; productName?: string } | null | undefined,
): string {
  const name = app?.productName ?? "";
  if (!name) return "(unnamed app)";
  switch (app?.type) {
    case "prod": return name;
    case "staging": return `${name} (Staging)`;
    case "dev": return `${name} (Dev)`;
    default: return name;
  }
}

export type ConfigExposure = "public" | "private" | "secret";

export type ConfigValueType =
  | "string"
  | "int"
  | "decimal"
  | "bool"
  | "string[]"
  | "int[]"
  | "decimal[]"
  | "bool[]"
  | "json";

type ConfigKeyStatus = "active" | "archived";

export type ConfigKey = {
  id: string;
  productId: string;

  // stable identifier used by SDKs/integrations
  key: string;

  // optional human help text
  description?: string | null;

  exposure: ConfigExposure;

  valueType: ConfigValueType;

  status: ConfigKeyStatus;

  createdAt: string;
  updatedAt: string;
  createdBy: string;
};

export type ConfigValue = {
  id: string;

  productId: string;
  appId: string;
  configKeyId: string;

  // for non-secret: value_json returned as JSON (decoded to JS)
  // for secret: omitted (write-only)
  value?: unknown | null;

  // for secret: true if set, omitted/false if unset
  hasSecret?: boolean;

  updatedAt: string;
  updatedBy: string;
};


export interface ProductMemberRole {
  id: string;
  productId: string;
  appId?: string | null; // null = project-wide (all apps)
  userId: string;
  roleId: string;
  createdAt: string;
}

export interface Role {
  id: string;
  productId: string;
  name: string;
  slug: string;
  permissions: Permission[];
  createdAt: string;
  updatedAt: string;
}

export type FeatureFlag = {
  id: string;
  productId: string;
  key: string;
  description?: string | null;
  defaultEnabled: boolean;
  scope: string,
  status: string;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
};

export type FeatureFlagOverride = {
  id: string;
  productId: string;
  appId: string;
  featureFlagId: string;
  enabled: boolean;
  roleIds?: string[];
  status: string;
  updatedAt: string;
  updatedBy: string;
};

export interface APIKey {
  id: string;
  name: string;
  prefix: string;
  createdAt: string;
}

// User Fields
export type UserFieldValueType = "string" | "bool" | "date";
export type UserFieldVisibility = "client" | "server";

export type UserField = {
  id: string;
  productId: string;
  key: string;
  valueType: UserFieldValueType;
  visibility: UserFieldVisibility;
  userEditable: boolean;
  label?: string | null;
  status: string;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
};

export interface CorsOrigin {
  id: string;
  appId: string;
  origin: string;
  createdAt: string;
}

export function isSafeRedirectURL(url: string): boolean {
  try {
    const u = new URL(url);
    return u.protocol === "https:" || u.protocol === "http:";
  } catch {
    return false;
  }
}