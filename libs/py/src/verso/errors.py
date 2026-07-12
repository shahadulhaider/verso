"""RFC 9457 Problem Details for HTTP APIs."""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True, slots=True)
class ProblemDetail:
    """RFC 9457 problem detail representation."""

    type: str
    title: str
    status: int
    detail: str
    instance: str = ""


def create_problem(
    status: int, title: str, detail: str, *, instance: str = ""
) -> ProblemDetail:
    """Create a ProblemDetail with standard type URI."""
    return ProblemDetail(
        type=f"https://httpstatuses.io/{status}",
        title=title,
        status=status,
        detail=detail,
        instance=instance,
    )


def problem_json_response(problem: ProblemDetail) -> dict:
    """Convert ProblemDetail to a dict suitable for FastAPI JSONResponse."""
    resp = {
        "type": problem.type,
        "title": problem.title,
        "status": problem.status,
        "detail": problem.detail,
    }
    if problem.instance:
        resp["instance"] = problem.instance
    return resp


# --- Standard factories ---


def not_found(detail: str = "Resource not found") -> ProblemDetail:
    """404 Not Found."""
    return create_problem(404, "Not Found", detail)


def bad_request(detail: str = "Invalid request") -> ProblemDetail:
    """400 Bad Request."""
    return create_problem(400, "Bad Request", detail)


def unauthorized(detail: str = "Authentication required") -> ProblemDetail:
    """401 Unauthorized."""
    return create_problem(401, "Unauthorized", detail)


def forbidden(detail: str = "Access denied") -> ProblemDetail:
    """403 Forbidden."""
    return create_problem(403, "Forbidden", detail)


def internal_error(detail: str = "Internal server error") -> ProblemDetail:
    """500 Internal Server Error."""
    return create_problem(500, "Internal Server Error", detail)


def conflict(detail: str = "Resource conflict") -> ProblemDetail:
    """409 Conflict."""
    return create_problem(409, "Conflict", detail)
