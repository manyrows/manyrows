import { Box, Stack, Typography } from "@mui/material";
import { useApp } from "../App.tsx";
import PageHeader from "../components/PageHeader.tsx";
import SigningKeysCard from "../profile/SigningKeysCard.tsx";

// SigningKeysPage hosts the JWT signing-key rotation panel at the
// workspace level. Super-admin only - non-super accounts see the
// "not authorised" state from SigningKeysCard itself, which keeps
// the auth check in one place.
export default function SigningKeysPage() {
  const app = useApp();
  const isSuper = !!app.appData.account?.isSuper;

  return (
    <Box>
      <Stack spacing={3} sx={{ maxWidth: 720 }}>
        <PageHeader
          title="JWT signing keys"
          subtitle="Rotate the ES256 keypair that signs end-user JWTs. New tokens sign with the new key; the previous key is kept in JWKS until you retire it, so tokens already in flight keep verifying."
          size={28}
          mb={0}
        />

        {!isSuper ? (
          <Typography variant="body2" color="text.secondary">
            Only super-admin accounts can rotate signing keys.
          </Typography>
        ) : (
          <SigningKeysCard isSuper={isSuper} />
        )}
      </Stack>
    </Box>
  );
}
