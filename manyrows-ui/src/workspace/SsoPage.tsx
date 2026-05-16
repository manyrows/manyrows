import { Box, Stack, Typography } from "@mui/material";
import { Fingerprint } from "lucide-react";
import PageHeader from "../components/PageHeader.tsx";
import EmptyState from "../components/EmptyState.tsx";
import StatusChip from "../components/StatusChip.tsx";

export default function SsoPage() {
  return (
    <Box>
      <PageHeader
        title="SSO"
        subtitle="Let a user sign in once and carry that session across the apps you choose."
        meta={<StatusChip label="Coming soon" severity="primary" />}
      />

      <EmptyState
        icon={<Fingerprint size={20} strokeWidth={1.5} />}
        title="Single sign-on is on the way"
        description={
          <Stack spacing={1} alignItems="center">
            <Typography variant="body2" color="text.secondary">
              You'll be able to pick a set of apps and let a user log in once for all of them — one session, shared identity, no re-prompt as they move between apps.
            </Typography>
          </Stack>
        }
        maxWidth={520}
      />
    </Box>
  );
}