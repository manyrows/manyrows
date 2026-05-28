import * as React from "react";
import {
  Box,
  Chip,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Typography,
} from "@mui/material";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import {
  ArrowLeft,
  ChevronDown,
  Settings,
  IdCard,
  Layers,
  ShieldCheck,
  ArrowLeftRight,
  Palette,
  Braces,
  Sparkles,
  Rocket,
  History,
  LogIn,
  Key,
  Lock,
  Users,
  Flag,
  SlidersHorizontal,
  Bell,
  LineChart,
} from "lucide-react";

import { appTypeLabel } from "../core.ts";

interface AppContext {
  appType?: string; // "dev" | "staging" | "prod"
  appBasePath: string; // e.g. "/app/workspace/123/products/456/apps/789"
  appPage: string; // current sub-page (e.g. "auth-methods")
  onOpenSwitcher?: (anchor: HTMLElement) => void;
}

interface Props {
  value: string;
  basePath: string; // e.g. "/app/workspace/123/products/456"
  workspaceBasePath: string; // e.g. "/app/workspace/123"
  app?: AppContext;
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

export default function ProductSideMenu({ value, basePath, workspaceBasePath, app }: Props) {
  const { t } = useTranslation();

  // When we're truly inside an app (appPage set), "Apps" is the active
  // product row. When the sub-tree is only sticky (user popped out to
  // Roles/Permissions/etc.) the real project page should highlight
  // instead.
  const insideApp = !!app && !!app.appPage;
  const effectiveValue = insideApp ? "apps" : value;

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
      <List disablePadding sx={{ display: "grid", gap: 0, mb: 0.5 }}>
        <ListItemButton
          component={Link}
          to={workspaceBasePath}
          sx={{ ...itemSx, color: "text.secondary" }}
        >
          <ListItemIcon sx={iconSx}>
            <ArrowLeft size={ICON_SIZE} strokeWidth={ICON_STROKE} />
          </ListItemIcon>
          <ListItemText
            primary={t("project.nav.backToWorkspace", { defaultValue: "Back to Workspace" })}
            primaryTypographyProps={{ fontSize: 12.5, fontWeight: 500 }}
          />
        </ListItemButton>
      </List>

      <Section label={t("section.project")}>
        <NavItem
          label={t("project.nav.apps")}
          value="apps"
          icon={<Layers size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={effectiveValue}
          basePath={basePath}
          trailing={
            app ? (
              <Box
                component="span"
                sx={{
                  display: "inline-flex",
                  alignItems: "center",
                  color: "text.disabled",
                  ml: 0.5,
                  flexShrink: 0,
                }}
              >
                <ChevronDown size={12} strokeWidth={ICON_STROKE} />
              </Box>
            ) : undefined
          }
        />

        {app && <AppSubNav app={app} t={t} />}

        <NavItem
          label={t("project.nav.appDiff")}
          value="appDiff"
          icon={<ArrowLeftRight size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={effectiveValue}
          basePath={basePath}
        />
        <NavItem
          label={t("project.nav.roles")}
          value="roles"
          icon={<IdCard size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={effectiveValue}
          basePath={basePath}
        />
        <NavItem
          label={t("project.nav.permissions")}
          value="permissions"
          icon={<ShieldCheck size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={effectiveValue}
          basePath={basePath}
        />
        <NavItem
          label={t("project.nav.branding")}
          value="branding"
          icon={<Palette size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={effectiveValue}
          basePath={basePath}
          trailing={
            <Box
              component="span"
              sx={{
                display: "inline-flex",
                alignItems: "center",
                color: "primary.main",
                ml: 0.5,
                flexShrink: 0,
              }}
            >
              <Sparkles
                size={11}
                strokeWidth={2}
                aria-label={t("branding.premium")}
              />
            </Box>
          }
        />
        <NavItem
          label={t("project.nav.schema")}
          value="schema"
          icon={<Braces size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={effectiveValue}
          basePath={basePath}
          trailing={
            <Box
              component="span"
              sx={{
                display: "inline-flex",
                alignItems: "center",
                color: "primary.main",
                ml: 0.5,
                flexShrink: 0,
              }}
            >
              <Sparkles
                size={11}
                strokeWidth={2}
                aria-label={t("schema.premium")}
              />
            </Box>
          }
        />
        <NavItem
          label={t("project.nav.settings")}
          value="settings"
          icon={<Settings size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          selected={effectiveValue}
          basePath={basePath}
        />
      </Section>
    </Box>
  );
}

// Sub-tree under "Apps" when an app is open. Indented with a thin left rail
// so the parent/child relationship reads at a glance.
function AppSubNav({ app, t }: { app: AppContext; t: (k: string, o?: Record<string, unknown>) => string }) {
  const { appType, appBasePath, appPage, onOpenSwitcher } = app;

  const appItems: { to: string; key: string; icon: React.ReactNode; label: string }[] = [
    { to: appBasePath, key: "appDetail", icon: <Settings size={ICON_SIZE} strokeWidth={ICON_STROKE} />, label: t("app.nav.settings", { defaultValue: "App Settings" }) },
    { to: `${appBasePath}/auth-methods`, key: "auth-methods", icon: <Lock size={ICON_SIZE} strokeWidth={ICON_STROKE} />, label: t("app.nav.authMethods", { defaultValue: "Auth Methods" }) },
    { to: `${appBasePath}/security`, key: "security", icon: <ShieldCheck size={ICON_SIZE} strokeWidth={ICON_STROKE} />, label: t("app.nav.security", { defaultValue: "Security" }) },
    { to: `${appBasePath}/api-keys`, key: "api-keys", icon: <Key size={ICON_SIZE} strokeWidth={ICON_STROKE} />, label: t("app.nav.apiKeys", { defaultValue: "API Keys" }) },
    { to: `${appBasePath}/webhooks`, key: "webhooks", icon: <Bell size={ICON_SIZE} strokeWidth={ICON_STROKE} />, label: t("app.nav.webhooks", { defaultValue: "Webhooks" }) },
    { to: `${appBasePath}/quick-start`, key: "quick-start", icon: <Rocket size={ICON_SIZE} strokeWidth={ICON_STROKE} />, label: t("app.nav.quickStart", { defaultValue: "Quick Start" }) },
  ];

  const dataItems: { to: string; key: string; icon: React.ReactNode; label: string }[] = [
    { to: `${appBasePath}/members`, key: "members", icon: <Users size={ICON_SIZE} strokeWidth={ICON_STROKE} />, label: t("app.nav.members", { defaultValue: "Users" }) },
    { to: `${appBasePath}/sessions`, key: "sessions", icon: <History size={ICON_SIZE} strokeWidth={ICON_STROKE} />, label: t("app.nav.sessions", { defaultValue: "Sessions" }) },
    { to: `${appBasePath}/auth-logs`, key: "auth-logs", icon: <LogIn size={ICON_SIZE} strokeWidth={ICON_STROKE} />, label: t("app.nav.authLogs", { defaultValue: "Auth Logs" }) },
    { to: `${appBasePath}/features`, key: "features", icon: <Flag size={ICON_SIZE} strokeWidth={ICON_STROKE} />, label: t("project.nav.features", { defaultValue: "Feature Flags" }) },
    { to: `${appBasePath}/config`, key: "config", icon: <SlidersHorizontal size={ICON_SIZE} strokeWidth={ICON_STROKE} />, label: t("project.nav.config", { defaultValue: "Config Keys" }) },
    { to: `${appBasePath}/insights`, key: "insights", icon: <LineChart size={ICON_SIZE} strokeWidth={ICON_STROKE} />, label: t("app.nav.insights", { defaultValue: "Insights" }) },
  ];

  return (
    <Box
      sx={{
        position: "relative",
        ml: 2.25,
        pl: 1,
        mt: 0.25,
        mb: 0.5,
        borderLeft: "1px solid",
        borderColor: "divider",
      }}
    >
      {/* Env chip + switcher anchor the sub-tree. The app name itself
          is the page-header job — repeating it here is noise. */}
      <Box
        sx={{
          px: 1,
          pt: 0.5,
          pb: 0.75,
          display: "flex",
          alignItems: "center",
          gap: 0.75,
          minWidth: 0,
        }}
      >
        {appType && (
          <Chip
            size="small"
            label={appTypeLabel({ type: appType })}
            variant="outlined"
            sx={{
              height: 18,
              fontSize: 9.5,
              fontWeight: 600,
              letterSpacing: "0.08em",
              fontFamily: "var(--font-mono)",
              textTransform: "uppercase",
              flexShrink: 0,
              "& .MuiChip-label": { px: 0.75 },
              ...(appType === "prod" && { borderColor: "error.main", color: "error.main" }),
              ...(appType === "staging" && { borderColor: "warning.main", color: "warning.main" }),
              ...(appType === "dev" && { borderColor: "success.main", color: "success.main" }),
            }}
          />
        )}
        {onOpenSwitcher && (
          <Box
            component="button"
            onClick={(e: React.MouseEvent<HTMLButtonElement>) => onOpenSwitcher(e.currentTarget)}
            sx={{
              display: "inline-flex",
              alignItems: "center",
              gap: 0.25,
              px: 0.5,
              height: 18,
              ml: appType ? "auto" : -0.5,
              border: "none",
              borderRadius: 0.5,
              bgcolor: "transparent",
              color: "text.disabled",
              fontFamily: "inherit",
              fontSize: 10.5,
              fontWeight: 500,
              cursor: "pointer",
              flexShrink: 0,
              transition: "background-color 120ms ease, color 120ms ease",
              "&:hover": { bgcolor: "action.hover", color: "text.primary" },
            }}
          >
            {t("projectHome.switch")}
            <ChevronDown size={9} strokeWidth={1.75} />
          </Box>
        )}
      </Box>

      <List disablePadding sx={{ display: "grid", gap: 0 }}>
        {appItems.map((it) => (
          <SubNavItem key={it.key} to={it.to} selected={appPage === it.key} icon={it.icon} label={it.label} />
        ))}
      </List>

      <Typography
        sx={{
          px: 1,
          pb: 0.5,
          pt: 1.25,
          display: "block",
          color: "text.disabled",
          fontFamily: "var(--font-mono)",
          fontSize: 9.5,
          letterSpacing: "0.16em",
          fontWeight: 500,
          textTransform: "uppercase",
        }}
      >
        {t("app.nav.dataSection", { defaultValue: "Data" })}
      </Typography>

      <List disablePadding sx={{ display: "grid", gap: 0 }}>
        {dataItems.map((it) => (
          <SubNavItem key={it.key} to={it.to} selected={appPage === it.key} icon={it.icon} label={it.label} />
        ))}
      </List>
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
  trailing,
}: {
  label: string;
  value: string;
  selected: string;
  basePath: string;
  icon: React.ReactNode;
  trailing?: React.ReactNode;
}) {
  const isSel = selected === value;
  return (
    <ListItemButton
      component={Link}
      to={`${basePath}/${value}`}
      selected={isSel}
      sx={itemSx}
    >
      <ListItemIcon sx={iconSx}>{icon}</ListItemIcon>
      <ListItemText
        primary={label}
        primaryTypographyProps={{
          fontSize: 12.5,
          fontWeight: isSel ? 600 : 500,
        }}
      />
      {trailing}
    </ListItemButton>
  );
}

function SubNavItem({
  to,
  selected,
  icon,
  label,
}: {
  to: string;
  selected: boolean;
  icon: React.ReactNode;
  label: string;
}) {
  return (
    <ListItemButton
      component={Link}
      to={to}
      selected={selected}
      sx={{ ...itemSx, px: 1 }}
    >
      <ListItemIcon sx={iconSx}>{icon}</ListItemIcon>
      <ListItemText
        primary={label}
        primaryTypographyProps={{
          fontSize: 12.5,
          fontWeight: selected ? 600 : 500,
        }}
      />
    </ListItemButton>
  );
}
