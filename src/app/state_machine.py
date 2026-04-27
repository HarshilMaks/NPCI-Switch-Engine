from collections.abc import Mapping


TERMINAL_STATES = {"COMPLETED", "FAILED", "REVERSED", "REVERSAL_FAILED"}

ALLOWED_TRANSITIONS: Mapping[str, set[str]] = {
    "INITIATED": {"AUTH_PENDING", "FAILED"},
    "AUTH_PENDING": {"AUTHORIZED", "FAILED"},
    "AUTHORIZED": {"DEBIT_POSTED", "FAILED"},
    "DEBIT_POSTED": {"CREDIT_POSTED", "REVERSAL_PENDING"},
    "CREDIT_POSTED": {"COMPLETED"},
    "REVERSAL_PENDING": {"REVERSED", "REVERSAL_FAILED"},
    "COMPLETED": set(),
    "FAILED": set(),
    "REVERSED": set(),
    "REVERSAL_FAILED": set(),
}


class InvalidTransitionError(ValueError):
    pass


def ensure_transition_allowed(current: str, target: str) -> None:
    allowed = ALLOWED_TRANSITIONS.get(current, set())
    if target not in allowed:
        raise InvalidTransitionError(f"Illegal transition: {current} -> {target}")

