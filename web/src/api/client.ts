import createClient from "openapi-fetch";
import type { paths } from "./types";

const API_BASE = "/api/v1";

export const api = createClient<paths>({
  baseUrl: API_BASE,
});

// Add auth token to requests
api.use({
  onRequest({ request }) {
    const token = localStorage.getItem("tofui_token");
    if (token) {
      request.headers.set("Authorization", `Bearer ${token}`);
    }
    return request;
  },
  onResponse({ response }) {
    if (response.status === 401) {
      localStorage.removeItem("tofui_token");
      window.location.href = "/login";
    }
    return response;
  },
});
