import * as React from "react";
import { useTranslation } from "react-i18next";
import type { Product, Workspace } from "../core.ts";
import { Box, Button, Chip, Paper, Stack, Typography } from "@mui/material";
import PageHeader from "../components/PageHeader.tsx";
import { Braces, Sparkles, Check, Mail } from "lucide-react";

const SUPPORT_EMAIL = "support@manyrows.com";

interface Props {
  project: Product;
  workspace?: Workspace;
}

// Schema mirrors the Branding pattern: a premium feature that isn't built
// yet, so the page is the upsell + concierge entry point. Customers who
// want custom data fields / schemas reach out and we'll prioritise it.
export default function Schema({ project }: Props) {
  const { t } = useTranslation();

  const features = [
    t("schema.feature.userFields", { defaultValue: "Custom fields on user records" }),
    t("schema.feature.appRecords", { defaultValue: "App-level records and tables" }),
    t("schema.feature.types", { defaultValue: "Typed fields (text, number, date, enum, JSON)" }),
    t("schema.feature.validation", { defaultValue: "Validation, required fields, and defaults" }),
    t("schema.feature.api", { defaultValue: "Schema available via the admin and S2S APIs" }),
  ];

  const mailtoHref = React.useMemo(() => {
    const subject = t("schema.mailSubject", {
      defaultValue: "Schema request — {{product}}",
      product: project.name,
    });
    const body = t("schema.mailBody", {
      defaultValue:
        "Hi ManyRows team,\n\nWe'd like custom schemas for \"{{product}}\". Here's what we're thinking:\n\n- Records / objects we need:\n- Fields and types:\n- How it'd be consumed (admin UI, SDK, API):\n- Anything else:\n\nThanks!",
      product: project.name,
    });
    return `mailto:${SUPPORT_EMAIL}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`;
  }, [project.name, t]);

  return (
    <Box>
      <PageHeader
        title={t("schema.title", { defaultValue: "Schema" })}
        subtitle={t("schema.subtitle", {
          defaultValue:
            "Define custom fields and data shapes for your users and app records.",
        })}
        action={
          <Chip
            size="small"
            icon={<Sparkles size={12} strokeWidth={2} />}
            label={t("schema.premium", { defaultValue: "Premium" })}
            sx={{
              height: 24,
              fontSize: 10.5,
              fontWeight: 600,
              letterSpacing: "0.08em",
              fontFamily: "var(--font-mono)",
              textTransform: "uppercase",
              bgcolor: "transparent",
              color: "primary.main",
              border: "1px solid",
              borderColor: "primary.main",
              "& .MuiChip-icon": { color: "primary.main", ml: 0.75 },
            }}
          />
        }
      />

      <Paper
        variant="outlined"
        sx={{ borderRadius: 2, p: { xs: 3, sm: 4 }, maxWidth: 720 }}
      >
        <Stack spacing={2.5}>
          <Stack direction="row" spacing={2} alignItems="flex-start">
            <Box
              sx={{
                width: 40,
                height: 40,
                borderRadius: 1.5,
                display: "grid",
                placeItems: "center",
                color: "primary.main",
                border: "1px solid",
                borderColor: "divider",
                bgcolor: "background.paper",
                flexShrink: 0,
              }}
            >
              <Braces size={20} strokeWidth={1.75} />
            </Box>
            <Box sx={{ minWidth: 0 }}>
              <Typography
                sx={{
                  fontFamily: "var(--font-serif)",
                  fontSize: 20,
                  fontWeight: 500,
                  letterSpacing: "-0.02em",
                  lineHeight: 1.3,
                  fontOpticalSizing: "auto",
                }}
              >
                {t("schema.heroTitle", {
                  defaultValue: "Custom schemas, coming soon",
                })}
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mt: 0.75, maxWidth: 520 }}>
                {t("schema.heroBody", {
                  defaultValue:
                    "Model the data you care about — custom fields on users, app-level records, validation, and types. Reach out and we'll prioritise it for your account.",
                })}
              </Typography>
            </Box>
          </Stack>

          <Stack component="ul" spacing={1} sx={{ listStyle: "none", m: 0, p: 0 }}>
            {features.map((f) => (
              <Stack key={f} component="li" direction="row" spacing={1.25} alignItems="center">
                <Box component="span" sx={{ display: "inline-flex", color: "primary.main", flexShrink: 0 }}>
                  <Check size={15} strokeWidth={2.25} />
                </Box>
                <Typography variant="body2" sx={{ color: "text.primary" }}>
                  {f}
                </Typography>
              </Stack>
            ))}
          </Stack>

          <Box>
            <Button
              variant="contained"
              href={mailtoHref}
              startIcon={<Mail size={15} strokeWidth={1.75} />}
              disableElevation
              sx={{ borderRadius: 2, textTransform: "none", fontWeight: 600 }}
            >
              {t("schema.contactCta", { defaultValue: "Contact us about schemas" })}
            </Button>
            <Typography variant="caption" color="text.disabled" sx={{ display: "block", mt: 1.25 }}>
              {t("schema.contactHint", {
                defaultValue: "Opens an email to {{email}} with your project details prefilled.",
                email: SUPPORT_EMAIL,
              })}
            </Typography>
          </Box>
        </Stack>
      </Paper>
    </Box>
  );
}
