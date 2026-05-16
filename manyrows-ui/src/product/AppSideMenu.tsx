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
  ArrowLeft,
  Rocket,
  Settings,
  History,
  LogIn,
  Key,
  Lock,
  ShieldCheck,
  Users,
  Flag,
  SlidersHorizontal,
  Bell,
  LineChart,
} from "lucide-react";

interface Props {
  projectBasePath: string; // e.g. "/app/workspace/123/products/456"
  appBasePath: string; // e.g. "/app/workspace/123/products/456/apps/789"
  value: string; // current sub-page
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

export default function AppSideMenu({ projectBasePath, appBasePath, value }: Props) {
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
      {/* Back to apps list */}
      <List disablePadding sx={{ display: "grid", gap: 0, mb: 0.5 }}>
        <ListItemButton
          component={Link}
          to={`${projectBasePath}/apps`}
          sx={{ ...itemSx, color: "text.secondary" }}
        >
          <ListItemIcon sx={iconSx}>
            <ArrowLeft size={ICON_SIZE} strokeWidth={ICON_STROKE} />
          </ListItemIcon>
          <ListItemText
            primary={t("app.nav.backToProduct", { defaultValue: "Back to Product" })}
            primaryTypographyProps={{ fontSize: 12.5, fontWeight: 500 }}
          />
        </ListItemButton>
      </List>

      {/* App section header - the app name lives in the canvas title
          block, so don't repeat it here. */}
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
        App
      </Typography>

      {/* Settings group - configuration the operator changes occasionally. */}
      <List disablePadding sx={{ display: "grid", gap: 0 }}>
        <NavItem
          to={appBasePath}
          selected={value === "appDetail"}
          icon={<Settings size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          label={t("app.nav.settings", { defaultValue: "App Settings" })}
        />
        <NavItem
          to={`${appBasePath}/auth-methods`}
          selected={value === "auth-methods"}
          icon={<Lock size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          label={t("app.nav.authMethods", { defaultValue: "Auth Methods" })}
        />
        <NavItem
          to={`${appBasePath}/security`}
          selected={value === "security"}
          icon={<ShieldCheck size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          label={t("app.nav.security", { defaultValue: "Security" })}
        />
        <NavItem
          to={`${appBasePath}/api-keys`}
          selected={value === "api-keys"}
          icon={<Key size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          label={t("app.nav.apiKeys", { defaultValue: "API Keys" })}
        />
        <NavItem
          to={`${appBasePath}/webhooks`}
          selected={value === "webhooks"}
          icon={<Bell size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          label={t("app.nav.webhooks", { defaultValue: "Webhooks" })}
        />
        <NavItem
          to={`${appBasePath}/quick-start`}
          selected={value === "quick-start"}
          icon={<Rocket size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          label={t("app.nav.quickStart", { defaultValue: "Quick Start" })}
        />
      </List>

      {/* Data group - runtime activity / per-end-user records. */}
      <Typography
        sx={{
          px: 1.25,
          pb: 0.5,
          pt: 1.75,
          display: "block",
          color: "text.disabled",
          fontFamily: "var(--font-mono)",
          fontSize: 10,
          letterSpacing: "0.16em",
          fontWeight: 500,
          textTransform: "uppercase",
        }}
      >
        Data
      </Typography>

      <List disablePadding sx={{ display: "grid", gap: 0 }}>
        <NavItem
          to={`${appBasePath}/members`}
          selected={value === "members"}
          icon={<Users size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          label={t("app.nav.members", { defaultValue: "Users" })}
        />
        <NavItem
          to={`${appBasePath}/sessions`}
          selected={value === "sessions"}
          icon={<History size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          label={t("app.nav.sessions", { defaultValue: "Sessions" })}
        />
        <NavItem
          to={`${appBasePath}/auth-logs`}
          selected={value === "auth-logs"}
          icon={<LogIn size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          label={t("app.nav.authLogs", { defaultValue: "Auth Logs" })}
        />
        <NavItem
          to={`${appBasePath}/features`}
          selected={value === "features"}
          icon={<Flag size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          label={t("project.nav.features", { defaultValue: "Feature Flags" })}
        />
        <NavItem
          to={`${appBasePath}/config`}
          selected={value === "config"}
          icon={<SlidersHorizontal size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          label={t("project.nav.config", { defaultValue: "Config Keys" })}
        />
        <NavItem
          to={`${appBasePath}/insights`}
          selected={value === "insights"}
          icon={<LineChart size={ICON_SIZE} strokeWidth={ICON_STROKE} />}
          label={t("app.nav.insights", { defaultValue: "Insights" })}
        />
      </List>
    </Box>
  );
}

function NavItem({
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
    <ListItemButton component={Link} to={to} selected={selected} sx={itemSx}>
      <ListItemIcon sx={iconSx}>{icon}</ListItemIcon>
      <ListItemText
        primary={label}
        primaryTypographyProps={{ fontSize: 12.5, fontWeight: selected ? 600 : 500 }}
      />
    </ListItemButton>
  );
}
