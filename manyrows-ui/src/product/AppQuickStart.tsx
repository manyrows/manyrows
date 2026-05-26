import * as React from "react";
import {
  Box,
  IconButton,
  Stack,
  Tab,
  Tabs,
  Tooltip,
  Typography,
} from "@mui/material";
import { Copy } from "lucide-react";
import PageHeader from "../components/PageHeader.tsx";
import { useSnackbar } from "notistack";
import { useTranslation } from "react-i18next";
import type { Product, Workspace } from "../core.ts";

interface Props {
  project: Product;
  workspace: Workspace;
  appId: string;
}

function CodeBlock({ title, code, onCopy }: { title: string; code: string; onCopy: () => void }) {
  const { t } = useTranslation();
  return (
    <Box sx={{ borderRadius: 2, overflow: "hidden", border: "1px solid", borderColor: "divider" }}>
      <Stack
        direction="row"
        justifyContent="space-between"
        alignItems="center"
        sx={{ px: 1.5, py: 0.75, bgcolor: "action.hover", borderBottom: "1px solid", borderColor: "divider" }}
      >
        <Typography
          sx={{
            fontFamily: "var(--font-mono)",
            textTransform: "uppercase",
            letterSpacing: "0.12em",
            fontSize: 10,
            fontWeight: 600,
            color: "text.disabled",
          }}
        >
          {title}
        </Typography>
        <Tooltip title={t("apps.copySnippet")}>
          <IconButton size="small" onClick={onCopy} sx={{ p: 0.25 }}>
            <Box component="span" sx={{ fontSize: 13, color: "text.disabled" }}><Copy size={14} strokeWidth={1.75} /></Box>
          </IconButton>
        </Tooltip>
      </Stack>
      <Box
        component="pre"
        sx={{
          m: 0,
          px: 1.75,
          py: 1.25,
          fontSize: 12,
          lineHeight: 1.65,
          fontFamily: "var(--font-mono)",
          color: "text.primary",
          overflow: "auto",
          bgcolor: "background.default",
        }}
      >
        {code}
      </Box>
    </Box>
  );
}

export default function AppQuickStart({ workspace, appId }: Props) {
  const { enqueueSnackbar } = useSnackbar();
  const { t } = useTranslation();

  // Use the install's own hostname in the snippets - that's the URL
  // the customer's app needs to point AppKit at.
  const baseURL = typeof window !== "undefined" && window.location?.origin
    ? window.location.origin
    : "https://auth.yourdomain.com";

  const minimal = `import { AppKit, AppKitAuthed } from "@manyrows/appkit-react";

export default function App() {
  return (
    <AppKit baseURL="${baseURL}" workspace="${workspace.slug}" appId="${appId}">
      <AppKitAuthed>
        <p>You are signed in!</p>
      </AppKitAuthed>
    </AppKit>
  );
}`;

  const full = `import {
  AppKit,
  AppKitAuthed,
  // useUser,
  // useToken,
  // useAuthFetch,
  // useRoles,
  // useRole,
  // usePermissions,
  // usePermission,
  // useFeatureFlag,
  // useFeatureFlags,
  // useConfigValue,
  // useConfig,
  // useSetPassword,
} from "@manyrows/appkit-react";

export default function App() {
  return (
    <AppKit
      baseURL="${baseURL}"
      workspace="${workspace.slug}"
      appId="${appId}"

      // --- Theme ---
      theme={{
        primaryColor: '#7a5c2e',
        backgroundColor: '#c8b890',
        // cardBackgroundColor: '#ffffff',
        // colorMode: 'auto',        // 'light' | 'dark' | 'auto'
        // fontFamily: 'Inter, sans-serif',
        // fontSize: '14px',
        // radiusSm: '4px',
        // radiusMd: '8px',
        // radiusLg: '12px',
        // shadowCard: '0 2px 8px rgba(0,0,0,0.1)',
        // cssOverrides: { '--my-var': '#000' },
      }}

      // --- Auth UI ---
      authHeader={
        <div style={{ textAlign: 'center', marginBottom: '1.5rem' }}>
          <p style={{ fontSize: '2rem', margin: 0 }}>My Brand</p>
        </div>
      }
      // initialScreen="login"       // 'login' | 'register' | 'forgot-password'
      // hideAuthUI={false}           // hide built-in login UI (build your own)
      // publicAccess={false}         // show children even when not authenticated

      // --- Auth Routes (SPA routing for auth screens) ---
      // authRoutes={{
      //   login: '/login',
      //   register: '/register',
      //   'forgot-password': '/forgot-password',
      // }}
      // authRedirect="/dashboard"    // redirect here after login

      // --- Label overrides ---
      // labels={{
      //   signInTitle: 'Welcome back',
      //   createAccountTitle: 'Get started',
      //   signIn: 'Log in',
      //   createAccount: 'Sign up',
      //   signInWithGoogle: 'Continue with Google',
      //   keepMeSignedIn: 'Remember me',
      // }}

      // --- Loading & Errors ---
      // loading={<MySpinner />}
      // hideLoadingUI={false}
      // errorUI={(err) => <MyError error={err} />}
      // hideErrorUI={false}

      // --- Callbacks ---
      onError={(e) => console.warn('AppKit error:', e)}
      // onReady={(info) => console.log('AppKit ready:', info.version)}
      // onState={(snapshot) => console.log('State:', snapshot?.status)}
      // onReadyState={(snapshot) => console.log('Authenticated:', snapshot.appData)}
      // onScreenChange={(screen) => console.log('Auth screen:', screen)}

      // --- Advanced ---
      // timeoutMs={4000}
      // debug={false}
      // silent={false}
    >
      <AppKitAuthed
        // fallback={<p>Please sign in</p>}
      >
        <MyApp />
      </AppKitAuthed>
    </AppKit>
  );
}

function MyApp() {
  // --- User & Auth ---
  // const user = useUser();           // { id, email }
  // const token = useToken();         // JWT string for API calls
  // const authFetch = useAuthFetch(); // fetch() with auto Bearer token

  // --- Roles & Permissions ---
  // const roles = useRoles();                 // ['admin', 'editor']
  // const isAdmin = useRole('admin');          // true/false
  // const permissions = usePermissions();      // ['read', 'write']
  // const canWrite = usePermission('write');   // true/false

  // --- Feature Flags ---
  // const flags = useFeatureFlags();           // [{ key, enabled }]
  // const betaOn = useFeatureFlag('beta');     // true/false

  // --- Config ---
  // const config = useConfig();                         // [{ key, type, value }]
  // const limit = useConfigValue('max_items', 100);     // typed value with fallback

  // --- Password ---
  // const setPassword = useSetPassword();
  // await setPassword({ password: 'newPassword123', currentPassword: 'oldOne' });

  return <div>Your app here</div>;
}`;

  const vanillaJs = `<!-- Add the AppKit script -->
<script src="${baseURL}/appkit/assets/appkit.js" defer></script>

<!-- Container for the auth UI -->
<div id="manyrows-app"></div>

<script>
  // Wait for the script to load
  window.addEventListener('load', () => {
    const handle = window.ManyRows.AppKit.init({
      containerId: 'manyrows-app',
      workspace: '${workspace.slug}',
      appId: '${appId}',

      // Called when auth state changes
      onState: (snapshot) => {
        if (snapshot.status === 'authenticated') {
          const token = snapshot.jwtToken;
          console.log('User:', snapshot.appData?.account?.email);
          console.log('Roles:', snapshot.appData?.roles);
          console.log('Permissions:', snapshot.appData?.permissions);
          console.log('JWT:', token);
          // Use the token for your API calls:
          // fetch('/api/data', { headers: { Authorization: 'Bearer ' + token } })
          document.getElementById('manyrows-app').style.display = 'none';
          document.getElementById('my-app').style.display = 'block';
        } else if (snapshot.status === 'unauthenticated') {
          document.getElementById('manyrows-app').style.display = 'block';
          document.getElementById('my-app').style.display = 'none';
        }
      },

      // Optional: customize theme
      // theme: { primaryColor: '#7a5c2e' },

      // Optional: error handler
      // onError: (err) => console.warn('AppKit error:', err),
    });

    // To log out: handle.logout()
  });
</script>

<div id="my-app" style="display: none;">
  <p>You are signed in!</p>
</div>`;

  const copy = (code: string) => async () => {
    try {
      await navigator.clipboard.writeText(code);
      enqueueSnackbar(t("apps.copied", { label: t("apps.quickStart") }), { variant: "success" });
    } catch {
      enqueueSnackbar(t("apps.copyFailed"), { variant: "error" });
    }
  };

  const [tab, setTab] = React.useState(0);

  return (
    <Stack spacing={3}>
      <PageHeader title={t("apps.quickStart")} mb={0} />

      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ borderBottom: 1, borderColor: "divider" }}>
        <Tab label="React" />
        <Tab label="Vanilla JS" />
      </Tabs>

      {tab === 0 && (
        <Stack spacing={3}>
          <Typography variant="body2" color="text.secondary">
            {t("app.quickStart.minimalDescription", { defaultValue: "The minimum you need to add authentication to your React app." })}
          </Typography>

          <CodeBlock title="App.tsx - Minimal" code={minimal} onCopy={copy(minimal)} />

          <Typography variant="body2" color="text.secondary">
            {t("app.quickStart.fullDescription", { defaultValue: "All available features with optional ones commented out. Uncomment what you need." })}
          </Typography>

          <CodeBlock title="App.tsx - All Features" code={full} onCopy={copy(full)} />
        </Stack>
      )}

      {tab === 1 && (
        <Stack spacing={3}>
          <Typography variant="body2" color="text.secondary">
            {t("app.quickStart.vanillaDescription", { defaultValue: "Use AppKit without React. Load the script directly and control auth with plain JavaScript." })}
          </Typography>

          <CodeBlock title="index.html" code={vanillaJs} onCopy={copy(vanillaJs)} />
        </Stack>
      )}
    </Stack>
  );
}
