import * as React from "react";
import type { UserField, UserFieldValueType, UserFieldVisibility } from "../core.ts";
import axios from "axios";
import { extractApiError } from "../lib/apiError.ts";
import { useSnackbar } from "notistack";
import { useTranslation } from "react-i18next";

import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  IconButton,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";
import PageHeader from "../components/PageHeader.tsx";
import EmptyState from "../components/EmptyState.tsx";
import { ClipboardList, Plus, RefreshCw, SquarePen, Trash2 } from "lucide-react";

interface Props {
  workspaceId: string;
  poolId: string;
  embedded?: boolean;
}

export default function UserFields({ workspaceId, poolId, embedded }: Props) {
  const { enqueueSnackbar } = useSnackbar();
  const { t } = useTranslation();

  const basePath = `/admin/workspace/${workspaceId}/userPools/${poolId}/userFields`;

  const [loading, setLoading] = React.useState(true);
  const [fields, setFields] = React.useState<UserField[]>([]);
  const [saving, setSaving] = React.useState(false);

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const res = await axios.get(basePath);
      setFields(res.data?.userFields ?? []);
    } catch {
      enqueueSnackbar(t("userFields.failedToLoad", { defaultValue: "Failed to load user fields" }), { variant: "error" });
    } finally {
      setLoading(false);
    }
  }, [basePath, enqueueSnackbar, t]);

  React.useEffect(() => { void load(); }, [load]);

  // Create dialog
  const [createOpen, setCreateOpen] = React.useState(false);
  const [createKey, setCreateKey] = React.useState("");
  const [createType, setCreateType] = React.useState<UserFieldValueType>("string");
  const [createVisibility, setCreateVisibility] = React.useState<UserFieldVisibility>("server");
  const [createUserEditable, setCreateUserEditable] = React.useState(false);
  const [createLabel, setCreateLabel] = React.useState("");

  const openCreate = () => {
    setCreateKey("");
    setCreateType("string");
    setCreateVisibility("server");
    setCreateUserEditable(false);
    setCreateLabel("");
    setCreateOpen(true);
  };

  const doCreate = async () => {
    const key = createKey.trim();
    if (!key) return;

    setSaving(true);
    try {
      await axios.post(basePath, {
        key,
        valueType: createType,
        visibility: createVisibility,
        userEditable: createUserEditable,
        label: createLabel.trim(),
      });
      setCreateOpen(false);
      enqueueSnackbar(t("userFields.created", { defaultValue: "User field created" }), { variant: "success" });
      void load();
    } catch (e) {
      enqueueSnackbar(extractApiError(e, "Failed to create"), { variant: "error" });
    } finally {
      setSaving(false);
    }
  };

  // Edit dialog
  const [editOpen, setEditOpen] = React.useState(false);
  const [editField, setEditField] = React.useState<UserField | null>(null);
  const [editVisibility, setEditVisibility] = React.useState<UserFieldVisibility>("server");
  const [editUserEditable, setEditUserEditable] = React.useState(false);
  const [editLabel, setEditLabel] = React.useState("");
  const [editStatus, setEditStatus] = React.useState("active");

  const openEdit = (f: UserField) => {
    setEditField(f);
    setEditVisibility(f.visibility);
    setEditUserEditable(f.userEditable);
    setEditLabel(f.label || "");
    setEditStatus(f.status);
    setEditOpen(true);
  };

  const doEdit = async () => {
    if (!editField) return;
    setSaving(true);
    try {
      await axios.patch(`${basePath}/${editField.id}`, {
        visibility: editVisibility,
        userEditable: editUserEditable,
        label: editLabel.trim() || null,
        status: editStatus,
      });
      setEditOpen(false);
      enqueueSnackbar(t("userFields.updated", { defaultValue: "User field updated" }), { variant: "success" });
      void load();
    } catch (e) {
      enqueueSnackbar(extractApiError(e, "Failed to update"), { variant: "error" });
    } finally {
      setSaving(false);
    }
  };

  // Delete
  const doDelete = async (f: UserField) => {
    if (!confirm(t("userFields.deleteConfirm", { key: f.key, defaultValue: `Delete field "${f.key}"? This cannot be undone.` }))) return;
    setSaving(true);
    try {
      await axios.delete(`${basePath}/${f.id}`);
      enqueueSnackbar(t("userFields.deleted", { defaultValue: "User field deleted" }), { variant: "success" });
      void load();
    } catch (e) {
      enqueueSnackbar(extractApiError(e, "Failed to delete"), { variant: "error" });
    } finally {
      setSaving(false);
    }
  };

  const activeFields = fields.filter((f) => f.status === "active");
  const archivedFields = fields.filter((f) => f.status === "archived");

  return (
    <Box>
      <Stack spacing={2.5}>
        {embedded ? (
          <Stack direction="row" alignItems="center" justifyContent="space-between">
            <Typography variant="body2" color="text.secondary">
              {t("userFields.descriptionPool", { defaultValue: "Define custom fields for users in this pool. Values are stored per user." })}
            </Typography>
            <Stack direction="row" spacing={0.5} alignItems="center">
              <Tooltip title={t("common.refresh", { defaultValue: "Refresh" })}>
                <span>
                  <IconButton size="small" onClick={() => void load()} disabled={loading}>
                    {loading ? <CircularProgress size={16} /> : <RefreshCw size={14} strokeWidth={1.75} />}
                  </IconButton>
                </span>
              </Tooltip>
              <Button
                size="small"
                variant="contained"
                disableElevation
                startIcon={<Plus size={14} strokeWidth={1.75} />}
                onClick={openCreate}
                disabled={saving}
                sx={{ borderRadius: 2, textTransform: "none" }}
              >
                {t("userFields.newField", { defaultValue: "New Field" })}
              </Button>
            </Stack>
          </Stack>
        ) : (
          <PageHeader
            title={t("userFields.title", { defaultValue: "User Fields" })}
            subtitle={t("userFields.descriptionPool", { defaultValue: "Define custom fields for users in this pool. Values are stored per user." })}
            mb={0}
            action={
              <>
                <Tooltip title={t("common.refresh", { defaultValue: "Refresh" })}>
                  <span>
                    <IconButton size="small" onClick={() => void load()} disabled={loading}>
                      {loading ? <CircularProgress size={16} /> : <RefreshCw size={14} strokeWidth={1.75} />}
                    </IconButton>
                  </span>
                </Tooltip>
                <Button
                  size="small"
                  variant="contained"
                  disableElevation
                  startIcon={<Plus size={14} strokeWidth={1.75} />}
                  onClick={openCreate}
                  disabled={saving}
                  sx={{ borderRadius: 2, textTransform: "none" }}
                >
                  {t("userFields.newField", { defaultValue: "New Field" })}
                </Button>
              </>
            }
          />
        )}

        {/* Table */}
        {activeFields.length > 0 ? (
          <TableContainer component={Paper} variant="outlined">
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell sx={{ fontWeight: 600, fontSize: 12 }}>Key</TableCell>
                  <TableCell sx={{ fontWeight: 600, fontSize: 12 }}>Type</TableCell>
                  <TableCell sx={{ fontWeight: 600, fontSize: 12 }}>Visibility</TableCell>
                  <TableCell sx={{ fontWeight: 600, fontSize: 12 }}>User Editable</TableCell>
                  <TableCell sx={{ fontWeight: 600, fontSize: 12 }}>Label</TableCell>
                  <TableCell sx={{ fontWeight: 600, fontSize: 12 }} align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {activeFields.map((f) => (
                  <TableRow key={f.id} hover>
                    <TableCell>
                      <Typography sx={{ fontSize: 13, fontWeight: 600, fontFamily: "var(--font-mono)" }}>{f.key}</Typography>
                    </TableCell>
                    <TableCell>
                      <Chip size="small" label={f.valueType} sx={{ height: 20, fontSize: 10 }} />
                    </TableCell>
                    <TableCell>
                      <Chip
                        size="small"
                        label={f.visibility === "client" ? "Client" : "Server"}
                        color={f.visibility === "client" ? "primary" : "default"}
                        sx={{ height: 20, fontSize: 10 }}
                      />
                    </TableCell>
                    <TableCell>
                      <Typography sx={{ fontSize: 13, color: f.userEditable ? "success.main" : "text.disabled" }}>
                        {f.userEditable ? "Yes" : "No"}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography sx={{ fontSize: 12, color: "text.secondary" }} noWrap>
                        {f.label || "-"}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Stack direction="row" spacing={0.5} justifyContent="flex-end">
                        <Tooltip title="Edit">
                          <span>
                            <IconButton size="small" onClick={() => openEdit(f)} disabled={saving}>
                              <SquarePen size={14} strokeWidth={1.75} />
                            </IconButton>
                          </span>
                        </Tooltip>
                        <Tooltip title="Delete">
                          <span>
                            <IconButton size="small" onClick={() => void doDelete(f)} disabled={saving}>
                              <Trash2 size={14} strokeWidth={1.75} />
                            </IconButton>
                          </span>
                        </Tooltip>
                      </Stack>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        ) : !loading ? (
          <EmptyState
            icon={<ClipboardList size={18} strokeWidth={1.75} />}
            title={t("userFields.noFields", { defaultValue: "No user fields defined yet." })}
            description={t("userFields.noFieldsDesc", { defaultValue: "Add custom fields like display name, role, or any per-user attributes your app needs." })}
            action={
              <Button
                size="small"
                variant="contained"
                disableElevation
                startIcon={<Plus size={14} strokeWidth={1.75} />}
                onClick={openCreate}
                sx={{ borderRadius: 2, textTransform: "none" }}
              >
                {t("userFields.createFirst", { defaultValue: "Create your first field" })}
              </Button>
            }
          />
        ) : null}

        {/* Archived fields */}
        {archivedFields.length > 0 && (
          <>
            <Typography variant="subtitle2" color="text.secondary" sx={{ mt: 2 }}>
              {t("userFields.archived", { defaultValue: "Archived" })} ({archivedFields.length})
            </Typography>
            <TableContainer component={Paper} variant="outlined" sx={{ borderRadius: 2, opacity: 0.6 }}>
              <Table size="small">
                <TableBody>
                  {archivedFields.map((f) => (
                    <TableRow key={f.id}>
                      <TableCell>
                        <Typography sx={{ fontSize: 13, fontFamily: "var(--font-mono)" }}>{f.key}</Typography>
                      </TableCell>
                      <TableCell>
                        <Chip size="small" label={f.valueType} sx={{ height: 20, fontSize: 10 }} />
                      </TableCell>
                      <TableCell>
                        <Chip size="small" label="Archived" sx={{ height: 20, fontSize: 10 }} />
                      </TableCell>
                      <TableCell align="right">
                        <Tooltip title="Edit">
                          <span>
                            <IconButton size="small" onClick={() => openEdit(f)} disabled={saving}>
                              <SquarePen size={14} strokeWidth={1.75} />
                            </IconButton>
                          </span>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </>
        )}
      </Stack>

      {/* Create Dialog */}
      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>{t("userFields.dialog.createTitle", { defaultValue: "New User Field" })}</DialogTitle>
        <Box
          component="form"
          onSubmit={(e) => { e.preventDefault(); if (!saving) void doCreate(); }}
        >
          <DialogContent sx={{ pt: 1 }}>
            <Stack spacing={2} sx={{ mt: 1 }}>
              <TextField
                size="small"
                label="Key"
                value={createKey}
                onChange={(e) => setCreateKey(e.target.value)}
                helperText="Unique identifier (e.g. first_name, plan_tier)"
                inputProps={{ spellCheck: false }}
                autoFocus
                disabled={saving}
              />

              <FormControl size="small" fullWidth>
                <InputLabel>Type</InputLabel>
                <Select label="Type" value={createType} onChange={(e) => setCreateType(e.target.value as UserFieldValueType)} disabled={saving}>
                  <MenuItem value="string">String</MenuItem>
                  <MenuItem value="bool">Boolean</MenuItem>
                  <MenuItem value="date">Date</MenuItem>
                </Select>
              </FormControl>

              <FormControl size="small" fullWidth>
                <InputLabel>Visibility</InputLabel>
                <Select label="Visibility" value={createVisibility} onChange={(e) => setCreateVisibility(e.target.value as UserFieldVisibility)} disabled={saving}>
                  <MenuItem value="client">
                    <Stack>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>Client</Typography>
                      <Typography variant="caption" color="text.secondary">Visible via AppKit SDK</Typography>
                    </Stack>
                  </MenuItem>
                  <MenuItem value="server">
                    <Stack>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>Server</Typography>
                      <Typography variant="caption" color="text.secondary">Admin and server SDK only</Typography>
                    </Stack>
                  </MenuItem>
                </Select>
              </FormControl>

              <FormControlLabel
                control={<Switch checked={createVisibility === "server" ? false : createUserEditable} onChange={(e) => setCreateUserEditable(e.target.checked)} disabled={saving || createVisibility === "server"} />}
                label={
                  <Stack>
                    <Typography variant="body2" sx={{ fontWeight: 500 }}>User Editable</Typography>
                    <Typography variant="caption" color="text.secondary">
                      {createVisibility === "server" ? "Server-only fields cannot be user-editable" : "Allow users to update this field via the client SDK"}
                    </Typography>
                  </Stack>
                }
                sx={{ alignItems: "flex-start", ml: 0 }}
              />

              <TextField
                size="small"
                label="Label"
                value={createLabel}
                onChange={(e) => setCreateLabel(e.target.value)}
                disabled={saving}
              />
            </Stack>
          </DialogContent>
          <DialogActions sx={{ px: 3, pb: 2 }}>
            <Button onClick={() => setCreateOpen(false)} disabled={saving}>Cancel</Button>
            <Button type="submit" variant="contained" disableElevation disabled={!createKey.trim() || !createLabel.trim() || saving}>
              Create
            </Button>
          </DialogActions>
        </Box>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={editOpen} onClose={() => setEditOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>{t("userFields.dialog.editTitle", { defaultValue: "Edit User Field" })}</DialogTitle>
        <Box
          component="form"
          onSubmit={(e) => { e.preventDefault(); if (!saving) void doEdit(); }}
        >
          <DialogContent sx={{ pt: 1 }}>
            <Stack spacing={2} sx={{ mt: 1 }}>
              {editField && (
                <Alert severity="info" sx={{ fontSize: 13 }}>
                  Key: <strong>{editField.key}</strong> - Type: <strong>{editField.valueType}</strong> (cannot be changed)
                </Alert>
              )}

              <FormControl size="small" fullWidth>
                <InputLabel>Visibility</InputLabel>
                <Select label="Visibility" value={editVisibility} onChange={(e) => setEditVisibility(e.target.value as UserFieldVisibility)} disabled={saving}>
                  <MenuItem value="client">Client</MenuItem>
                  <MenuItem value="server">Server</MenuItem>
                </Select>
              </FormControl>

              <FormControlLabel
                control={<Switch checked={editVisibility === "server" ? false : editUserEditable} onChange={(e) => setEditUserEditable(e.target.checked)} disabled={saving || editVisibility === "server"} />}
                label={editVisibility === "server" ? "User Editable (disabled for server-only fields)" : "User Editable"}
              />

              <TextField
                size="small"
                label="Label"
                value={editLabel}
                onChange={(e) => setEditLabel(e.target.value)}
                disabled={saving}
              />

              <FormControl size="small" fullWidth>
                <InputLabel>Status</InputLabel>
                <Select label="Status" value={editStatus} onChange={(e) => setEditStatus(e.target.value)} disabled={saving}>
                  <MenuItem value="active">Active</MenuItem>
                  <MenuItem value="archived">Archived</MenuItem>
                </Select>
              </FormControl>
            </Stack>
          </DialogContent>
          <DialogActions sx={{ px: 3, pb: 2 }}>
            <Button onClick={() => setEditOpen(false)} disabled={saving}>Cancel</Button>
            <Button type="submit" variant="contained" disableElevation disabled={saving}>Save</Button>
          </DialogActions>
        </Box>
      </Dialog>
    </Box>
  );
}
