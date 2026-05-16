import * as React from "react";
import axios from "axios";
import { extractApiError } from "../lib/apiError.ts";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Stack,
  Tooltip,
  Typography,
} from "@mui/material";
import { Copy, KeyRound, RefreshCw, Trash2 } from "lucide-react";
// JWT signing-key rotation panel. Super-admin only. Lives in the
// profile screen because there's no separate "install settings"
// surface today - keeps the route count low.
//
// Maps to three endpoints (manyrows-core/api/securityHandler.go):
//   GET    /admin/security/signing-keys
//   POST   /admin/security/signing-keys/rotate
//   POST   /admin/security/signing-keys/retire-previous
//
// The card stays mounted but renders null for non-super accounts so
// the parent Profile.tsx doesn't need to conditional-include it.

interface SigningKeyInfo {
  kid: string;
}

interface SigningKeyStatus {
  current: SigningKeyInfo;
  previous?: SigningKeyInfo | null;
}

export default function SigningKeysCard({ isSuper }: { isSuper: boolean }) {
  const [status, setStatus] = React.useState<SigningKeyStatus | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [rotateOpen, setRotateOpen] = React.useState(false);
  const [retireOpen, setRetireOpen] = React.useState(false);
  const [busy, setBusy] = React.useState(false);

  const fetchStatus = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await axios.get<SigningKeyStatus>("/admin/security/signing-keys");
      setStatus(res.data);
    } catch (e) {
      setError(extractApiError(e, "Failed to load signing keys"));
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    if (isSuper) {
      fetchStatus();
    }
  }, [isSuper, fetchStatus]);

  if (!isSuper) {
    return null;
  }

  async function doRotate() {
    setBusy(true);
    setError(null);
    try {
      const res = await axios.post<SigningKeyStatus>("/admin/security/signing-keys/rotate");
      setStatus(res.data);
      setRotateOpen(false);
    } catch (e) {
      setError(extractApiError(e, "Rotation failed"));
    } finally {
      setBusy(false);
    }
  }

  async function doRetire() {
    setBusy(true);
    setError(null);
    try {
      const res = await axios.post<SigningKeyStatus>("/admin/security/signing-keys/retire-previous");
      setStatus(res.data);
      setRetireOpen(false);
    } catch (e) {
      setError(extractApiError(e, "Retirement failed"));
    } finally {
      setBusy(false);
    }
  }

  function copy(text: string) {
    navigator.clipboard.writeText(text).catch(() => {
      /* silent - clipboard permission denied */
    });
  }

  return (
    <Card variant="outlined">
      <CardContent sx={{ p: { xs: 2, sm: 3 } }}>
        <Stack spacing={2}>
          <Stack direction="row" alignItems="flex-start" spacing={1.5}>
            <Box sx={{ flex: 1, minWidth: 0 }}>
              <Typography
                sx={{
                  display: "inline-flex",
                  alignItems: "center",
                  gap: 1,
                  fontFamily: "var(--font-mono)",
                  textTransform: "uppercase",
                  letterSpacing: "0.14em",
                  fontSize: 10,
                  fontWeight: 500,
                  color: "text.disabled",
                  mb: 0.5,
                }}
              >
                <KeyRound size={9} strokeWidth={1.75} />
                Cryptographic keys
              </Typography>
              <Typography sx={{ fontSize: 16, fontWeight: 600, letterSpacing: "-0.005em" }}>
                JWT signing keys
              </Typography>
            </Box>
            <Chip
              label="Super-admin"
              size="small"
              color="warning"
              variant="outlined"
              sx={{
                fontFamily: "var(--font-mono)",
                textTransform: "uppercase",
                letterSpacing: "0.08em",
                fontSize: 9.5,
                fontWeight: 600,
                height: 20,
              }}
            />
          </Stack>

          <Typography variant="body2" color="text.secondary">
            Rotate the ES256 keypair that signs end-user JWTs. New tokens sign with the
            new key; the previous key is kept in JWKS until you retire it, so tokens
            already in flight keep verifying. Wait at least the longest live
            refresh-token TTL (7 days by default, up to 30 days with remember-me, or
            whatever per-app override is set) before retiring the previous key.
          </Typography>

          {error && (
            <Alert severity="error">
              {error}
            </Alert>
          )}

          {loading && !status ? (
            <Box sx={{ display: "flex", justifyContent: "center", py: 2 }}>
              <CircularProgress size={24} />
            </Box>
          ) : status ? (
            <Stack spacing={1.5}>
              <KeyRow label="Current" kid={status.current.kid} onCopy={() => copy(status.current.kid)} />
              {status.previous && (
                <KeyRow
                  label="Previous"
                  kid={status.previous.kid}
                  onCopy={() => copy(status.previous!.kid)}
                  faded
                />
              )}
            </Stack>
          ) : null}

          <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
            <Button
              variant="outlined"
              startIcon={<RefreshCw size={14} strokeWidth={1.75} />}
              onClick={() => setRotateOpen(true)}
              disabled={busy || loading}
            >
              Rotate
            </Button>
            <Button
              variant="outlined"
              color="error"
              startIcon={<Trash2 size={14} strokeWidth={1.75} />}
              onClick={() => setRetireOpen(true)}
              disabled={busy || loading || !status?.previous}
            >
              Retire previous
            </Button>
          </Stack>
        </Stack>
      </CardContent>

      <Dialog
        open={rotateOpen}
        onClose={() => !busy && setRotateOpen(false)}
        fullWidth
        maxWidth="sm"
       
      >
        <DialogTitle>Rotate signing key?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            A new ES256 keypair will be generated and start signing new JWTs immediately.
            The current key moves into the "previous" slot and stays in JWKS so in-flight
            tokens keep verifying. After you've waited long enough for those tokens to
            expire (≥ longest refresh-token TTL), retire the previous key to complete
            the rotation.
          </DialogContentText>
          <DialogContentText sx={{ mt: 2 }}>
            If you have multiple replicas behind a load balancer, only this replica
            reloads its in-memory keyset - redeploy after rotating so every replica
            picks up the new kid on the issuance path.
          </DialogContentText>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2, gap: 1 }}>
          <Button onClick={() => setRotateOpen(false)} disabled={busy}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="warning"
            onClick={doRotate}
            disabled={busy}
            startIcon={busy ? <CircularProgress size={16} /> : <RefreshCw size={14} strokeWidth={1.75} />}
           
          >
            Rotate now
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog
        open={retireOpen}
        onClose={() => !busy && setRetireOpen(false)}
        fullWidth
        maxWidth="sm"
       
      >
        <DialogTitle>Retire previous key?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            The previous key will be deleted from the database and dropped from JWKS.
            Any JWT still signed with it will fail verification with "unknown kid",
            forcing the user to re-authenticate. Only do this once you're confident
            every refresh-token issued before the rotation has expired.
          </DialogContentText>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2, gap: 1 }}>
          <Button onClick={() => setRetireOpen(false)} disabled={busy}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            onClick={doRetire}
            disabled={busy}
            startIcon={busy ? <CircularProgress size={16} /> : <Trash2 size={14} strokeWidth={1.75} />}
           
          >
            Retire
          </Button>
        </DialogActions>
      </Dialog>
    </Card>
  );
}

function KeyRow({
  label,
  kid,
  onCopy,
  faded = false,
}: {
  label: string;
  kid: string;
  onCopy: () => void;
  faded?: boolean;
}) {
  return (
    <Stack
      direction="row"
      alignItems="center"
      spacing={1.5}
      sx={{
        opacity: faded ? 0.75 : 1,
        py: 1,
        px: 1.5,
        borderRadius: 2,
        bgcolor: "action.hover",
      }}
    >
      <Typography
        variant="caption"
        sx={{ fontWeight: 600, textTransform: "uppercase", letterSpacing: 0.5, minWidth: 64 }}
      >
        {label}
      </Typography>
      <Typography variant="body2" sx={{ fontFamily: "var(--font-mono)", flex: 1, overflow: "hidden", textOverflow: "ellipsis" }}>
        {kid}
      </Typography>
      <Tooltip title="Copy kid">
        <Button
          size="small"
          onClick={onCopy}
          sx={{ minWidth: 0, p: 0.5 }}
          aria-label="Copy kid to clipboard"
        >
          <Copy size={14} strokeWidth={1.75} />
        </Button>
      </Tooltip>
    </Stack>
  );
}
