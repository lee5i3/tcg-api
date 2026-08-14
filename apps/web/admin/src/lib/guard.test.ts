import { describe, expect, it } from "vitest";
import { guardTarget } from "./guard";

describe("guardTarget", () => {
  it("redirects unauthenticated visitors to /login", () => {
    expect(guardTarget(null, "/")).toBe("/login");
    expect(guardTarget(null, "/g/pokemon")).toBe("/login");
    expect(guardTarget(null, "/g/pokemon/s/base")).toBe("/login");
  });

  it("lets unauthenticated visitors see the login page", () => {
    expect(guardTarget(null, "/login")).toBeNull();
  });

  it("bounces authenticated admins from /login to the dashboard", () => {
    expect(guardTarget("secret", "/login")).toBe("/");
  });

  it("lets authenticated admins through everywhere else", () => {
    expect(guardTarget("secret", "/")).toBeNull();
    expect(guardTarget("secret", "/g/pokemon")).toBeNull();
    expect(guardTarget("secret", "/g/pokemon/s/base")).toBeNull();
  });
});
