"""JWT validation with JWKS key fetching."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

import httpx
from jose import jwt as jose_jwt
from jose.exceptions import ExpiredSignatureError, JWTError


@dataclass(frozen=True, slots=True)
class Claims:
    """Decoded JWT claims."""

    user_id: str
    roles: list[str]
    sub: str
    iss: str
    exp: int


class JWKSValidator:
    """Validates JWTs against a JWKS endpoint."""

    def __init__(self, jwks_url: str) -> None:
        self.jwks_url = jwks_url
        self._cached_keys: dict[str, Any] = {}

    async def _fetch_keys(self) -> dict[str, Any]:
        """Fetch JWKS from the configured endpoint."""
        if not self._cached_keys:
            async with httpx.AsyncClient() as client:
                resp = await client.get(self.jwks_url)
                resp.raise_for_status()
                self._cached_keys = resp.json()
        return self._cached_keys

    def _invalidate_cache(self) -> None:
        """Clear cached JWKS keys."""
        self._cached_keys = {}

    async def validate(self, token: str) -> Claims:
        """Validate a JWT token and return decoded claims.

        Raises:
            JWTError: If the token is invalid or signature verification fails.
            ExpiredSignatureError: If the token has expired.
        """
        jwks = await self._fetch_keys()

        try:
            payload = jose_jwt.decode(
                token,
                jwks,
                algorithms=["RS256"],
                options={"verify_aud": False},
            )
        except ExpiredSignatureError:
            self._invalidate_cache()
            raise
        except JWTError:
            # Try refetching keys once in case of rotation
            self._invalidate_cache()
            jwks = await self._fetch_keys()
            payload = jose_jwt.decode(
                token,
                jwks,
                algorithms=["RS256"],
                options={"verify_aud": False},
            )

        return Claims(
            user_id=payload.get("user_id", payload.get("sub", "")),
            roles=payload.get("roles", []),
            sub=payload.get("sub", ""),
            iss=payload.get("iss", ""),
            exp=payload.get("exp", 0),
        )
