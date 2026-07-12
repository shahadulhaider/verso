"""Tests for verso.jwt module."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from jose import jwt as jose_jwt
from jose.exceptions import ExpiredSignatureError, JWTError

from verso.jwt import Claims, JWKSValidator


def test_jwks_validator_init():
    """JWKSValidator stores the JWKS URL."""
    v = JWKSValidator("https://auth.example.com/.well-known/jwks.json")
    assert v.jwks_url == "https://auth.example.com/.well-known/jwks.json"
    assert v._cached_keys == {}


def test_claims_dataclass():
    """Claims holds decoded JWT fields."""
    c = Claims(user_id="u1", roles=["admin"], sub="u1", iss="verso", exp=9999)
    assert c.user_id == "u1"
    assert c.roles == ["admin"]
    assert c.sub == "u1"
    assert c.iss == "verso"
    assert c.exp == 9999


@pytest.mark.asyncio
async def test_validate_decodes_token():
    """validate fetches JWKS and decodes token into Claims."""
    fake_jwks = {"keys": [{"kty": "RSA", "kid": "1"}]}
    fake_payload = {
        "user_id": "user-42",
        "roles": ["reader"],
        "sub": "user-42",
        "iss": "verso-identity",
        "exp": 9999999999,
    }

    v = JWKSValidator("https://auth.test/.well-known/jwks.json")

    mock_response = MagicMock()
    mock_response.json.return_value = fake_jwks
    mock_response.raise_for_status = MagicMock()

    mock_client = AsyncMock()
    mock_client.get.return_value = mock_response
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)

    with patch("verso.jwt.httpx.AsyncClient", return_value=mock_client), \
         patch("verso.jwt.jose_jwt.decode", return_value=fake_payload):
        claims = await v.validate("fake.jwt.token")

    assert claims.user_id == "user-42"
    assert claims.roles == ["reader"]
    assert claims.sub == "user-42"
    assert claims.iss == "verso-identity"


@pytest.mark.asyncio
async def test_validate_caches_keys():
    """validate caches JWKS after first fetch."""
    fake_jwks = {"keys": []}
    fake_payload = {"sub": "u1", "exp": 9999}

    v = JWKSValidator("https://auth.test/jwks")
    v._cached_keys = fake_jwks  # Pre-cache

    with patch("verso.jwt.jose_jwt.decode", return_value=fake_payload):
        claims = await v.validate("cached.jwt.token")

    assert claims.sub == "u1"
    # No HTTP call should have been made since keys were cached


@pytest.mark.asyncio
async def test_validate_retries_on_jwt_error():
    """validate refetches keys once on JWTError then retries decode."""
    fake_jwks = {"keys": [{"kty": "RSA", "kid": "2"}]}
    fake_payload = {"sub": "u2", "user_id": "u2", "roles": [], "iss": "v", "exp": 99}

    v = JWKSValidator("https://auth.test/jwks")

    mock_response = MagicMock()
    mock_response.json.return_value = fake_jwks
    mock_response.raise_for_status = MagicMock()

    mock_client = AsyncMock()
    mock_client.get.return_value = mock_response
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)

    call_count = 0

    def mock_decode(token, keys, **kwargs):
        nonlocal call_count
        call_count += 1
        if call_count == 1:
            raise JWTError("bad sig")
        return fake_payload

    with patch("verso.jwt.httpx.AsyncClient", return_value=mock_client), \
         patch("verso.jwt.jose_jwt.decode", side_effect=mock_decode):
        claims = await v.validate("retry.jwt.token")

    assert claims.sub == "u2"
    assert call_count == 2


@pytest.mark.asyncio
async def test_validate_raises_expired():
    """validate re-raises ExpiredSignatureError and invalidates cache."""
    v = JWKSValidator("https://auth.test/jwks")
    v._cached_keys = {"keys": []}

    with patch("verso.jwt.jose_jwt.decode", side_effect=ExpiredSignatureError("expired")):
        with pytest.raises(ExpiredSignatureError):
            await v.validate("expired.token")

    assert v._cached_keys == {}  # Cache invalidated


def test_invalidate_cache():
    """_invalidate_cache clears cached keys."""
    v = JWKSValidator("https://auth.test/jwks")
    v._cached_keys = {"keys": [{"kid": "1"}]}
    v._invalidate_cache()
    assert v._cached_keys == {}
