import {useApp} from "../App.tsx";
import {useParams} from "react-router-dom";
import ProductHome from "./ProductHome.tsx";
import {Box} from "@mui/material";
import { useTranslation } from "react-i18next";

export default function ProductRouter() {
  const app = useApp();
  const params = useParams();
  const { t } = useTranslation();
  const workspaceId = params["workspaceId"]
  const productId = params["productId"]
  const appId = params["appId"]
  const appPage = params["appPage"]
  const ws = app.appData.workspaces.find(w => w.id === workspaceId)
  if (!ws) {
    return <Box sx={{ p: 2 }}>{t("projectRouter.noWorkspaceSelected")}</Box>
  }
  if (!productId) {
    return <Box sx={{ p: 2 }}>{t("projectRouter.noProductSelected")}</Box>
  }
  // The standalone "Product" summary page was removed: it only listed
  // apps, which the Apps page does in full. The product now lands
  // directly on Apps, and any old "/home" bookmark redirects there.
  const projectPage = params["projectPage"] === "home" ? "apps" : params["projectPage"]
  const page = appId ? "appDetail" : (projectPage ?? "apps")
  return <ProductHome page={page} productId={productId} workspace={ws} appId={appId} appPage={appPage} />
}