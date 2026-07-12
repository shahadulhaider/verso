import { describe, it, expect, vi } from "vitest";
import {
  createProblem,
  sendProblem,
  notFound,
  badRequest,
  unauthorized,
  forbidden,
  internalError,
  conflict,
} from "./index.js";

describe("createProblem", () => {
  it("creates a ProblemDetail with required fields", () => {
    const p = createProblem(404, "Not Found");
    expect(p).toEqual({
      type: "https://httpstatuses.com/404",
      title: "Not Found",
      status: 404,
    });
  });

  it("includes detail when provided", () => {
    const p = createProblem(400, "Bad Request", "missing field: name");
    expect(p.detail).toBe("missing field: name");
  });

  it("omits detail when undefined", () => {
    const p = createProblem(500, "Internal Server Error");
    expect(p).not.toHaveProperty("detail");
  });
});

describe("sendProblem", () => {
  it("sets status, content-type header, and sends JSON body", () => {
    const reply = {
      status: vi.fn().mockReturnThis(),
      header: vi.fn().mockReturnThis(),
      send: vi.fn().mockReturnThis(),
    };

    const problem = createProblem(422, "Unprocessable Entity", "invalid ISBN");
    sendProblem(reply, problem);

    expect(reply.status).toHaveBeenCalledWith(422);
    expect(reply.header).toHaveBeenCalledWith("content-type", "application/problem+json");
    expect(reply.send).toHaveBeenCalledWith(problem);
  });
});

describe("standard factories", () => {
  it("notFound", () => {
    const p = notFound("book not found");
    expect(p.status).toBe(404);
    expect(p.title).toBe("Not Found");
    expect(p.detail).toBe("book not found");
  });

  it("badRequest", () => {
    expect(badRequest().status).toBe(400);
  });

  it("unauthorized", () => {
    expect(unauthorized().status).toBe(401);
  });

  it("forbidden", () => {
    expect(forbidden().status).toBe(403);
  });

  it("internalError", () => {
    expect(internalError().status).toBe(500);
  });

  it("conflict", () => {
    expect(conflict("duplicate ISBN").status).toBe(409);
    expect(conflict("duplicate ISBN").detail).toBe("duplicate ISBN");
  });
});
