"""Tests for verso.errors module."""

from verso.errors import (
    ProblemDetail,
    bad_request,
    conflict,
    create_problem,
    forbidden,
    internal_error,
    not_found,
    problem_json_response,
    unauthorized,
)


def test_create_problem_fields():
    """create_problem returns ProblemDetail with correct fields."""
    p = create_problem(422, "Validation Error", "Name is required", instance="/users/123")
    assert p.status == 422
    assert p.title == "Validation Error"
    assert p.detail == "Name is required"
    assert p.instance == "/users/123"
    assert p.type == "https://httpstatuses.io/422"


def test_create_problem_default_instance():
    """create_problem defaults instance to empty string."""
    p = create_problem(400, "Bad Request", "oops")
    assert p.instance == ""


def test_problem_detail_is_frozen():
    """ProblemDetail is immutable."""
    p = create_problem(404, "Not Found", "gone")
    try:
        p.status = 500  # type: ignore[misc]
        assert False, "Should have raised"
    except AttributeError:
        pass


def test_not_found_factory():
    """not_found returns 404 ProblemDetail."""
    p = not_found()
    assert p.status == 404
    assert p.title == "Not Found"
    assert p.detail == "Resource not found"


def test_not_found_custom_detail():
    """not_found accepts custom detail."""
    p = not_found("User 42 not found")
    assert p.detail == "User 42 not found"


def test_bad_request_factory():
    """bad_request returns 400."""
    p = bad_request()
    assert p.status == 400
    assert p.title == "Bad Request"


def test_unauthorized_factory():
    """unauthorized returns 401."""
    p = unauthorized()
    assert p.status == 401
    assert p.title == "Unauthorized"


def test_forbidden_factory():
    """forbidden returns 403."""
    p = forbidden()
    assert p.status == 403
    assert p.title == "Forbidden"


def test_internal_error_factory():
    """internal_error returns 500."""
    p = internal_error()
    assert p.status == 500
    assert p.title == "Internal Server Error"


def test_conflict_factory():
    """conflict returns 409."""
    p = conflict()
    assert p.status == 409
    assert p.title == "Conflict"


def test_problem_json_response_structure():
    """problem_json_response returns dict with RFC 9457 fields."""
    p = create_problem(404, "Not Found", "gone")
    resp = problem_json_response(p)
    assert resp["type"] == "https://httpstatuses.io/404"
    assert resp["title"] == "Not Found"
    assert resp["status"] == 404
    assert resp["detail"] == "gone"
    assert "instance" not in resp  # omitted when empty


def test_problem_json_response_with_instance():
    """problem_json_response includes instance when set."""
    p = create_problem(400, "Bad", "bad", instance="/foo")
    resp = problem_json_response(p)
    assert resp["instance"] == "/foo"
