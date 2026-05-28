import * as React from "react";
import {
  Box,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Typography,
} from "@mui/material";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import {
  FolderKanban,
  Users,
  Settings,
  UserCog,
  Mail,
  KeyRound,
  Cookie,
  Boxes,
  Fingerprint,
} from "lucide-react";

interface Props {
  /** Active nav value - maps to the workspacePage URL segment. */
  value: string;
  /** Workspace base path, e.g. "/app/workspace/123". */
  basePath: string;
}

const itemSx = {
  px: 1.25,
  py: 0.35,
  minHeight: 28,
  borderRadius: 1,
};

const iconSx = {
  minWidth: 24,
  color: "text.secondary" as const,
  display: "flex",
  alignItems: "center",
};

const ICON_SIZE = 14;
const ICON_STROKE = 1.75;

// WorkspaceSideMenu surfaces every workspace-level page in a left
// rail so Settings / Team / Email aren't hidden behind a single
// "Settings" button. Mirrors ProductSideMenu density
// and styling so the chrome reads as one piece across all three
// admin levels.
export default function WorkspaceSideMenu({ value, basePath }: Props) {
  const { t } = useTranslation();

  return (
    <Box
      sx={{
        p: 1,
        borderRight: { md: "1px solid rgba(13, 10, 8, 0.06)" },
        bgcolor: { md: "background.default" },
        height: { md: "calc(100vh - 52px)" },
        position: { md: "sticky" },
        top: { md: 52 },
        overflowY: { md: "auto" },
        "&::-webkit-scrollbar": { width: 6 },
        "&::-webkit-scrollbar-thumb": { bgcolor: "rgba(13,10,8,0.10)", borderRadius: 3 },
      }}
    >
      <Section label={t("section.workspace", { defaultValue: "Workspace" })}>
        <NavItem
          label={t("workspace.nav.products", { defaultValue: "Products" })}
          value="home"
          icon={<FolderKanban size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={value}
          basePath={basePath}
        />
        <NavItem
          label={t("workspace.nav.users", { defaultValue: "Users" })}
          value="users"
          icon={<Users size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={value}
          basePath={basePath}
        />
        <NavItem
          label={t("workspace.nav.userPools", { defaultValue: "User pools" })}
          value="userPools"
          icon={<Boxes size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={value}
          basePath={basePath}
        />
        <NavItem
          label={t("workspace.nav.sso", { defaultValue: "SSO" })}
          value="sso"
          icon={<Fingerprint size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={value}
          basePath={basePath}
        />
      </Section>

      <Section label={t("section.settings", { defaultValue: "Settings" })}>
        <NavItem
          label={t("workspace.nav.general", { defaultValue: "General" })}
          value="settings"
          icon={<Settings size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={value}
          basePath={basePath}
        />
        <NavItem
          label={t("workspace.nav.team", { defaultValue: "Team" })}
          value="team"
          icon={<UserCog size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={value}
          basePath={basePath}
        />
        <NavItem
          label={t("workspace.nav.email", { defaultValue: "Email" })}
          value="emailSettings"
          icon={<Mail size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={value}
          basePath={basePath}
        />
      </Section>

      <Section label={t("section.security", { defaultValue: "Security" })}>
        <NavItem
          label={t("workspace.nav.cookieDomain", { defaultValue: "Cookie domain" })}
          value="cookieDomain"
          icon={<Cookie size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={value}
          basePath={basePath}
        />
        <NavItem
          label={t("workspace.nav.signingKeys", { defaultValue: "JWT signing keys" })}
          value="signingKeys"
          icon={<KeyRound size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={value}
          basePath={basePath}
        />
      </Section>
    </Box>
  );
}

function Section({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <>
      <Typography
        sx={{
          px: 1.25,
          pb: 0.5,
          pt: 1.25,
          display: "block",
          color: "text.disabled",
          fontFamily: "var(--font-mono)",
          fontSize: 10,
          letterSpacing: "0.16em",
          fontWeight: 500,
          textTransform: "uppercase",
        }}
      >
        {label}
      </Typography>
      <List disablePadding sx={{ display: "grid", gap: 0 }}>
        {children}
      </List>
    </>
  );
}

function NavItem({
  label,
  value,
  selected,
  basePath,
  icon,
}: {
  label: string;
  value: string;
  selected: string;
  basePath: string;
  icon: React.ReactNode;
}) {
  const isSel = selected === value;
  // "home" doesn't append a suffix - it's the bare workspace path.
  const to = value === "home" ? basePath : `${basePath}/${value}`;
  return (
    <ListItemButton component={Link} to={to} selected={isSel} sx={itemSx}>
      <ListItemIcon sx={iconSx}>{icon}</ListItemIcon>
      <ListItemText
        primary={label}
        primaryTypographyProps={{
          fontSize: 12.5,
          fontWeight: isSel ? 600 : 500,
        }}
      />
    </ListItemButton>
  );
}
