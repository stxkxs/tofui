import { create } from "zustand";
import type { User } from "@/api/types";

interface AuthState {
  token: string | null;
  user: User | null;
  isAuthenticated: boolean;
  setAuth: (token: string, user: User) => void;
  setUser: (user: User) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem("tofui_token"),
  user: null,
  isAuthenticated: !!localStorage.getItem("tofui_token"),

  setAuth: (token, user) => {
    localStorage.setItem("tofui_token", token);
    set({ token, user, isAuthenticated: true });
  },

  setUser: (user) => {
    set({ user });
  },

  logout: () => {
    localStorage.removeItem("tofui_token");
    set({ token: null, user: null, isAuthenticated: false });
  },
}));
